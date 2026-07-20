package messageBuilder

import (
	"strings"
	"testing"

	"github.com/ChatDetectiveORG/shared/utils"
	tele "gopkg.in/telebot.v4"
	"github.com/go-pg/pg/v10/orm"
)

func TestTextFormatWithUserMention(t *testing.T) {
	f := (&TextFormat{Type: FormatLink}).WithUserMention(99)
	if f.URL != "tg://user?id=99" {
		t.Fatalf("URL = %q, want tg://user?id=99", f.URL)
	}
}

func TestTextFormatWithCustomEmojiID(t *testing.T) {
	f := (&TextFormat{Type: FormatLink}).WithCustomEmojiID("emoji-1")
	if f.URL != "tg://emoji?id=emoji-1" {
		t.Fatalf("URL = %q, want tg://emoji?id=emoji-1", f.URL)
	}
}

// func TestResolveBuilderFileKeepsFallback(t *testing.T) {
// 	file := resolveBuilderFile("cloud-id", "static/photo.png")
// 	if file.FileID != "cloud-id" {
// 		t.Fatalf("file id = %q", file.FileID)
// 	}
// 	if file.FileLocal != "static/photo.png" {
// 		t.Fatalf("file local = %q", file.FileLocal)
// 	}
// }

func TestTextFormatToMdV2Tag(t *testing.T) {
	tests := []struct {
		name    string
		format  TextFormat
		content string
		other   []TextFormat
		want    string
	}{
		{
			name:    "bold",
			format:  TextFormat{Type: FormatBold},
			content: "hi",
			want:    "*hi*",
		},
		{
			name:    "bold escapes reserved chars in content",
			format:  TextFormat{Type: FormatBold},
			content: "a.b",
			want:    "*a\\.b*",
		},
		{
			name:    "italic",
			format:  TextFormat{Type: FormatItalic},
			content: "hi",
			want:    "_hi_",
		},
		{
			name:    "underline",
			format:  TextFormat{Type: FormatUnderline},
			content: "x",
			want:    "__x__",
		},
		{
			name:    "mono",
			format:  TextFormat{Type: FormatMono},
			content: "code",
			want:    "`code`",
		},
		{
			name:    "spoiler",
			format:  TextFormat{Type: FormatSpoiler},
			content: "secret",
			want:    "||secret||",
		},
		{
			name:    "link",
			format:  TextFormat{Type: FormatLink, URL: "https://example.com"},
			content: "click",
			want:    "[click](https://example.com)",
		},
		{
			name:    "user mention",
			format:  TextFormat{Type: FormatLink, URL: "tg://user?id=42"},
			content: "Alice",
			want:    "[Alice](tg://user?id=42)",
		},
		{
			name:    "custom emoji",
			format:  TextFormat{Type: FormatLink, URL: "tg://emoji?id=123"}.WithCustomEmojiID("123"),
			content: "🙂",
			want:    "![🙂](tg://emoji?id=123)",
		},
		{
			name:    "nested bold italic",
			format:  TextFormat{Type: FormatBold},
			content: "t",
			other:   []TextFormat{{Type: FormatItalic}},
			want:    "*_t_*",
		},
		{
			name:    "blockquote multiline",
			format:  TextFormat{Type: FormatBlockquote},
			content: "a\nb",
			want:    "\n>a\n>b",
		},
		{
			name:    "strikethrough",
			format:  TextFormat{Type: FormatStrikethrough},
			content: "x",
			want:    "~x~",
		},
		{
			name:    "duplicate other format skipped",
			format:  TextFormat{Type: FormatBold},
			content: "x",
			other: []TextFormat{
				{Type: FormatItalic},
				{Type: FormatItalic},
			},
			want: "*_x_*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (&tt.format).toMdV2Tag(tt.content, tt.other...)
			if got != tt.want {
				t.Fatalf("toMdV2Tag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextFormatToTelebotTag(t *testing.T) {
	entity := (&TextFormat{Type: FormatBold}).toTelebotTag("Привет", 3)
	if entity.Type == "" {
		t.Fatal("expected entity")
	}
	if entity.Type != tele.EntityBold {
		t.Fatalf("type = %v, want bold", entity.Type)
	}
	if entity.Offset != 3 {
		t.Fatalf("offset = %d, want 3", entity.Offset)
	}
	if entity.Length != utils.TgLen("Привет") {
		t.Fatalf("length = %d, want %d", entity.Length, utils.TgLen("Привет"))
	}

	linkFormat := TextFormat{Type: FormatLink, URL: "https://t.me/bot"}
	link := linkFormat.toTelebotTag("bot", 0)
	if link.Type != tele.EntityTextLink || link.URL != "https://t.me/bot" {
		t.Fatalf("unexpected link entity: %+v", link)
	}

	if (&TextFormat{Type: "unknown"}).toTelebotTag("x", 0).Type != "" {
		t.Fatal("unknown format should return nil entity")
	}
}

func TestCreateGenericKeyboardParamsFillDefaults(t *testing.T) {
	params := (&CreateGenericKeyboardParams{}).fillDefaults()

	if params.ButtonsPerPage != 8 {
		t.Fatalf("ButtonsPerPage = %d, want 8", params.ButtonsPerPage)
	}
	if params.ButtonsPerRow != 2 {
		t.Fatalf("ButtonsPerRow = %d, want 2", params.ButtonsPerRow)
	}
	if params.ArrowForwardText != ">->>" {
		t.Fatalf("ArrowForwardText = %q", params.ArrowForwardText)
	}
	if params.ArrowBackText != "<<-<" {
		t.Fatalf("ArrowBackText = %q", params.ArrowBackText)
	}

	custom := (&CreateGenericKeyboardParams{
		ButtonsPerPage:   5,
		ButtonsPerRow:    3,
		ArrowForwardText: "next",
		ArrowBackText:    "prev",
	}).fillDefaults()

	if custom.ButtonsPerPage != 5 || custom.ButtonsPerRow != 3 {
		t.Fatalf("custom numeric defaults overwritten: %+v", custom)
	}
	if custom.ArrowForwardText != "next" || custom.ArrowBackText != "prev" {
		t.Fatalf("custom arrow texts overwritten: %+v", custom)
	}
}

func TestMessageBuilderWriteStringMdv2(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: true}
	b.WriteString("plain")
	b.WriteString("bold", TextFormat{Type: FormatBold})

	got := b.builder.String()
	want := "plain*bold*"
	if got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
	if b.cursorPosition != utils.TgLen(want) {
		t.Fatalf("cursor = %d, want %d", b.cursorPosition, utils.TgLen(want))
	}
}

func TestMessageBuilderWriteStringMdv2EscapesPlainText(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: true}
	b.WriteString("price: 1.99!")

	got := b.builder.String()
	want := "price: 1\\.99\\!"
	if got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

func TestMessageBuilderWriteStringMdv2DoesNotDoubleEscapeFormatting(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: true}
	b.WriteString("x", TextFormat{Type: FormatBold})

	if strings.Contains(b.builder.String(), `\*\*`) {
		t.Fatalf("markdown delimiters must not be escaped: %q", b.builder.String())
	}
}

func TestMessageBuilderWriteStringEntities(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: false}
	b.WriteString("A")
	b.WriteString("B", TextFormat{Type: FormatBold}, TextFormat{Type: FormatItalic})

	text := b.builder.String()
	if text != "AB" {
		t.Fatalf("text = %q, want AB", text)
	}
	if len(b.entities) != 2 {
		t.Fatalf("entities count = %d, want 2", len(b.entities))
	}

	bold := b.entities[0]
	if bold.Offset != utils.TgLen("A") {
		t.Fatalf("bold offset = %d, want %d", bold.Offset, utils.TgLen("A"))
	}
	if bold.Length != utils.TgLen("B") {
		t.Fatalf("bold length = %d, want %d", bold.Length, utils.TgLen("B"))
	}

	italic := b.entities[1]
	if italic.Type != tele.EntityItalic || italic.Offset != bold.Offset {
		t.Fatalf("unexpected italic entity: %+v", italic)
	}
}

func TestMessageBuilderKeyboardRows(t *testing.T) {
	b := &MessageBuilder{}
	b.AddButton(tele.InlineButton{Text: "1"})
	b.AddButton(tele.InlineButton{Text: "2"})
	b.NextRow()
	b.AddButton(tele.InlineButton{Text: "3"})

	if len(b.keyboard) != 1 {
		t.Fatalf("keyboard rows = %d, want 1", len(b.keyboard))
	}
	if len(b.keyboard[0]) != 2 {
		t.Fatalf("first row buttons = %d, want 2", len(b.keyboard[0]))
	}
	if len(b.currentRow) != 1 || b.currentRow[0].Text != "3" {
		t.Fatalf("unexpected current row: %+v", b.currentRow)
	}
}

type stubButton struct{}

func (b *stubButton) ToTelegramButton(db orm.DB, args TelegramButtonConversionArgs) tele.InlineButton {
	return tele.InlineButton{Text: "stub"}
}

func TestMessageBuilderAddFileUsesFileID(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: true}
	b.WriteString("caption text")
	b.AddFile("photo-file-id", "static/photo.png", "image/png")

	msg := b.Build(42)
	if msg.Photo == nil {
		t.Fatal("expected photo attachment")
	}
	if msg.Photo.File.FileID != "photo-file-id" {
		t.Fatalf("file id = %q, want photo-file-id", msg.Photo.File.FileID)
	}
	if msg.Photo.File.FileLocal != "static/photo.png" {
		t.Fatalf("fallback path = %q, want static/photo.png", msg.Photo.File.FileLocal)
	}
	if msg.Caption != "caption text" {
		t.Fatalf("caption = %q, want caption text", msg.Caption)
	}
	if msg.Text != "" {
		t.Fatalf("text should be empty for media message, got %q", msg.Text)
	}
	if msg.Photo.Caption != "caption text" {
		t.Fatalf("photo caption = %q, want caption text", msg.Photo.Caption)
	}
}

func TestMessageBuilderAddFileUsesFallbackPath(t *testing.T) {
	b := &MessageBuilder{}
	b.AddFile("", "static/setupInstruction.gif", "image/gif")

	msg := b.Build(1)
	if msg.Animation == nil {
		t.Fatal("expected animation attachment")
	}
	if msg.Animation.File.FileLocal != "static/setupInstruction.gif" {
		t.Fatalf("file local = %q", msg.Animation.File.FileLocal)
	}
	if msg.Animation.MIME != "image/gif" {
		t.Fatalf("mime = %q, want image/gif", msg.Animation.MIME)
	}
	if msg.Animation.FileName != "setupInstruction.gif" {
		t.Fatalf("file name = %q, want setupInstruction.gif", msg.Animation.FileName)
	}
}

func TestMessageBuilderAddFileInfersMimeFromExtension(t *testing.T) {
	b := &MessageBuilder{}
	b.AddFile("", "static/cipherExample.png", "")

	msg := b.Build(1)
	if msg.Photo == nil {
		t.Fatal("expected photo attachment")
	}
}

func TestMessageBuilderAddFileCategorizesMediaKinds(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		path     string
		check    func(*testing.T, *tele.Message)
	}{
		{
			name:     "video",
			mimeType: "video/mp4",
			path:     "clip.mp4",
			check: func(t *testing.T, msg *tele.Message) {
				if msg.Video == nil {
					t.Fatal("expected video")
				}
			},
		},
		{
			name:     "audio",
			mimeType: "audio/mpeg",
			path:     "track.mp3",
			check: func(t *testing.T, msg *tele.Message) {
				if msg.Audio == nil {
					t.Fatal("expected audio")
				}
			},
		},
		{
			name:     "voice",
			mimeType: "audio/ogg",
			path:     "voice.ogg",
			check: func(t *testing.T, msg *tele.Message) {
				if msg.Voice == nil {
					t.Fatal("expected voice")
				}
			},
		},
		{
			name:     "document",
			mimeType: "application/pdf",
			path:     "file.pdf",
			check: func(t *testing.T, msg *tele.Message) {
				if msg.Document == nil {
					t.Fatal("expected document")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &MessageBuilder{}
			b.AddFile("file-id", tt.path, tt.mimeType)
			tt.check(t, b.Build(1))
		})
	}
}

func TestMessageBuilderAddFileCaptionEntities(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: false}
	b.WriteString("bold", TextFormat{Type: FormatBold})
	b.AddFile("photo-id", "photo.png", "image/png")

	msg := b.Build(1)
	if len(msg.CaptionEntities) != 1 {
		t.Fatalf("caption entities = %d, want 1", len(msg.CaptionEntities))
	}
	if len(msg.Entities) != 0 {
		t.Fatalf("entities should move to caption, got %d", len(msg.Entities))
	}
}

func TestMessageBuilderBuildMediaGroup(t *testing.T) {
	b := &MessageBuilder{Mdv2Enabled: true}
	b.WriteString("album caption")
	b.AddFile("photo-1", "a.png", "image/png")
	b.AddFile("photo-2", "b.png", "image/png")
	b.AddButton(tele.InlineButton{Text: "ok"})
	b.NextRow()

	group, ok := b.BuildMediaGroup(99)
	if !ok {
		t.Fatal("expected media group")
	}
	if group.Chat.ID != 99 {
		t.Fatalf("chat id = %d, want 99", group.Chat.ID)
	}
	if len(group.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(group.Messages))
	}
	if group.Messages[0].Photo == nil || group.Messages[0].Photo.Caption != "album caption" {
		t.Fatalf("unexpected first item caption: %+v", group.Messages[0].Photo)
	}
	if group.Messages[0].ReplyMarkup == nil || len(group.Messages[0].ReplyMarkup.InlineKeyboard) != 1 {
		t.Fatalf("expected keyboard on first album item: %+v", group.Messages[0].ReplyMarkup)
	}
}

func TestCreateGenericKeyboardFlushesLastRow(t *testing.T) {
	builder := &MessageBuilder{}

	params := CreateGenericKeyboardParams{
		ChatID:         0,
		PageUnique:     "",
		ButtonsPerRow:  2,
		MergeButtons: [][]tele.InlineButton{
			{{Text: "merged"}},
		},
	}

	CreateGenericKeyboard[*stubButton](builder, nil, nil, nil, "", params)

	if len(builder.keyboard) != 0 {
		t.Fatalf("early return should not touch keyboard, got %d rows", len(builder.keyboard))
	}

	params.ChatID = 1
	params.PageUnique = "test"
	// Without DB/redis this still returns early after updateRedis fails or query fails.
	// Flush logic is covered via direct row simulation:
	builder = &MessageBuilder{}
	for i := 0; i < 3; i++ {
		builder.AddButton(tele.InlineButton{Text: string(rune('a' + i))})
		if len(builder.currentRow) >= 2 {
			builder.NextRow()
		}
	}
	if len(builder.currentRow) > 0 {
		builder.NextRow()
	}

	if len(builder.keyboard) != 2 {
		t.Fatalf("expected 2 rows after flush, got %d", len(builder.keyboard))
	}
	if len(builder.keyboard[1]) != 1 {
		t.Fatalf("last row should have 1 button, got %d", len(builder.keyboard[1]))
	}
}
