package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── color palette ──────────────────────────────────────────────────────────────
// A cohesive dark theme built around deep indigo / slate tones.
var (
	clrBg              = lipgloss.Color("234") // deep slate frame
	clrSurface         = lipgloss.Color("236") // main panel fill
	clrSurfaceAlt      = lipgloss.Color("237") // raised chrome and headers
	clrSurfaceElevated = lipgloss.Color("239") // modal / selected emphasis
	clrAccent          = lipgloss.Color("111") // soft electric blue accent
	clrAccentFg        = lipgloss.Color("255") // bright text on accent
	clrDir             = lipgloss.Color("68")  // darker blue for directories
	clrDirHidden       = lipgloss.Color("60")  // dimmer blue for hidden directories
	clrFile            = lipgloss.Color("255") // bright white for normal files
	clrFileHidden      = lipgloss.Color("245") // dim white for hidden files
	clrExec            = lipgloss.Color("150") // sage green for executables
	clrMedia           = lipgloss.Color("221") // warm amber for media
	clrDoc             = lipgloss.Color("189") // pale lilac for docs
	clrConfig          = lipgloss.Color("223") // sand for config files
	clrBinary          = lipgloss.Color("210") // coral for binary / unknown
	clrSize            = lipgloss.Color("248") // soft steel for metadata
	clrMuted           = lipgloss.Color("245") // secondary text
	clrDim             = lipgloss.Color("240") // dividers / low contrast text
	clrBreadcrumb      = lipgloss.Color("189") // breadcrumb path text
	clrPathSep         = lipgloss.Color("243") // breadcrumb separators
	clrHintKey         = lipgloss.Color("117") // footer keycaps
	clrHintText        = lipgloss.Color("250") // footer descriptions
	clrStatus          = lipgloss.Color("189") // normal status copy
	clrBorder          = lipgloss.Color("241") // default panel border
	clrBorderStrong    = lipgloss.Color("111") // active border
	clrTitle           = lipgloss.Color("255") // bright panel titles
	clrLoading         = lipgloss.Color("221") // loading indicator
	clrScrollbar       = lipgloss.Color("110") // scroll indicator
	clrDanger          = lipgloss.Color("203") // destructive accent
	clrDangerSoft      = lipgloss.Color("52")  // destructive surface
)

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
}

// nerdFonts controls whether Nerd Font glyphs are used.
// Set SEER_NO_NERD_FONT=1 to force plain Unicode fallback.
var nerdFonts = os.Getenv("SEER_NO_NERD_FONT") != "1"

// nerdIconByExt maps file extensions to specific Nerd Font glyphs.
var nerdIconByExt = map[string]string{
	// languages
	".go":    "\ue627 ",  //
	".js":    "\ue60c ",  //
	".ts":    "\ue628 ",  //
	".jsx":   "\ue60c ",  //
	".tsx":   "\ue60c ",  //
	".py":    "\ue606 ",  //
	".rb":    "\ue21e ",  //
	".rs":    "\ue7a8 ",  //
	".c":     "\ue61e ",  //
	".cpp":   "\ue61d ",  //
	".h":     "\uf0fd ",  //
	".java":  "\ue204 ",  //
	".cs":    "\uf031b ", // 󰌛
	".php":   "\ue60a ",  //
	".swift": "\ue755 ",  //
	".kt":    "\ue634 ",  //
	".lua":   "\ue620 ",  //
	".hs":    "\ue61f ",  //
	".vim":   "\ue62b ",  //
	".sh":    "\uf489 ",  //
	".bash":  "\uf489 ",  //
	".zsh":   "\uf489 ",  //
	".fish":  "\uf489 ",  //
	".ps1":   "\uf489 ",  //
	".bat":   "\uf489 ",  //
	".cmd":   "\uf489 ",  //
	// docs
	".md":       "\ue609 ", //
	".markdown": "\ue609 ", //
	".mdx":      "\ue609 ", //
	".rst":      "\uf15c ", //
	".txt":      "\uf15c ", //
	// config
	".json": "\ue60b ",  //
	".yaml": "\uf481 ",  //
	".yml":  "\uf481 ",  //
	".toml": "\uf481 ",  //
	".xml":  "\uf05c0 ", // 󰗀
	".env":  "\uf462 ",  //
	".ini":  "\uf17a ",  //
	".conf": "\uf17a ",  //
	// images
	".png":  "\uf1c5 ", //
	".jpg":  "\uf1c5 ", //
	".jpeg": "\uf1c5 ", //
	".gif":  "\uf1c5 ", //
	".webp": "\uf1c5 ", //
	".svg":  "\uf1c5 ", //
	".bmp":  "\uf1c5 ", //
	// misc
	".mmd":          "\ueb43 ", //
	".mermaid":      "\ueb43 ", //
	".pdf":          "\uf1c1 ", //
	".zip":          "\uf410 ", //
	".tar":          "\uf410 ", //
	".gz":           "\uf410 ", //
	".gitignore":    "\ue702 ", //
	".dockerignore": "\uf308 ", //
}

// nerdIconByCategory is the fallback Nerd Font icon per broad category.
var nerdIconByCategory = map[fileCategory]string{
	catDir:    "\uf07b ", //
	catImage:  "\uf1c5 ", //
	catDoc:    "\uf15c ", //
	catCode:   "\uf121 ", //
	catConfig: "\uf462 ", //
	catExec:   "\uf489 ", //
	catBinary: "\uf471 ", //
}

// plainIcon is the Unicode-only fallback per category.
var plainIcon = map[fileCategory]string{
	catDir:    "▸ ",
	catImage:  "⬡ ",
	catDoc:    "≡ ",
	catCode:   "⟨⟩ ",
	catConfig: "⚙ ",
	catExec:   "⚡ ",
	catBinary: "⬟ ",
}

func categorise(e entry) fileCategory {
	if e.isDir {
		return catDir
	}
	ext := strings.ToLower(filepath.Ext(e.name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".tiff":
		return catImage
	case ".md", ".markdown", ".mdx", ".rst", ".txt":
		return catDoc
	case ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd":
		return catExec
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".rs", ".c", ".cpp",
		".h", ".java", ".cs", ".php", ".swift", ".kt",
		".lua", ".ex", ".exs", ".hs", ".ml", ".mli", ".clj", ".scala",
		".vim", ".mmd", ".mermaid":
		return catCode
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".env", ".conf", ".config",
		".xml", ".dockerignore", ".gitignore", ".editorconfig", ".eslintrc",
		".prettierrc", ".babelrc", ".nvmrc":
		return catConfig
	}
	return catOther
}

func fileIcon(cat fileCategory) string {
	return fileIconExt(cat, "")
}

func fileIconExt(cat fileCategory, ext string) string {
	if !nerdFonts {
		if icon, ok := plainIcon[cat]; ok {
			return icon
		}
		return "· "
	}
	if ext != "" {
		if icon, ok := nerdIconByExt[strings.ToLower(ext)]; ok {
			return icon
		}
	}
	if icon, ok := nerdIconByCategory[cat]; ok {
		return icon
	}
	return "\uf15b " // generic file
}

func fileColor(cat fileCategory) lipgloss.Style {
	switch cat {
	case catDir:
		return lipgloss.NewStyle().Foreground(clrDir).Bold(true)
	case catImage:
		return lipgloss.NewStyle().Foreground(clrMedia)
	case catDoc:
		return lipgloss.NewStyle().Foreground(clrDoc)
	case catCode:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("231"))
	case catConfig:
		return lipgloss.NewStyle().Foreground(clrConfig)
	case catExec:
		return lipgloss.NewStyle().Foreground(clrExec)
	case catBinary:
		return lipgloss.NewStyle().Foreground(clrBinary)
	default:
		return lipgloss.NewStyle().Foreground(clrFile)
	}
}

func isHiddenName(name string) bool {
	return strings.HasPrefix(name, ".")
}

func entryNameStyle(e entry) lipgloss.Style {
	switch {
	case e.isDir && isHiddenName(e.name):
		return lipgloss.NewStyle().Foreground(clrDirHidden).Bold(true)
	case e.isDir:
		return lipgloss.NewStyle().Foreground(clrDir).Bold(true)
	case isHiddenName(e.name):
		return lipgloss.NewStyle().Foreground(clrFileHidden)
	default:
		return fileColor(categorise(e))
	}
}
