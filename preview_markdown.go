package main

// seerMarkdownStyle is a custom glamour JSON style that matches seer's dark
// theme and produces a rich, polished markdown preview:
//
//   - H1: warm cream text on indigo background (banner-style)
//   - H2: electric blue, bold, "── " prefix
//   - H3: lavender, bold
//   - H4–H6: progressively dimmer grays
//   - Strong: bright white bold
//   - Emph: pale lilac italic
//   - Inline code: warm yellow on dark-gray background
//   - Code blocks: dark background with nord-palette chroma highlighting
//   - Blockquotes: gray italic with │ marker
//   - Links: cyan underlined; link text bold blue
//   - Tables: unicode box separators
//
// document.margin=1 → output width = wordWrap−2 (matches our width+2 setting).
const seerMarkdownStyle = `{
  "document": {
    "block_suffix": "\n",
    "color":        "252",
    "margin":       1
  },
  "block_quote": {
    "indent":       1,
    "indent_token": "│ ",
    "color":        "246",
    "italic":       true
  },
  "list": {
    "color":        "252",
    "level_indent": 2
  },
  "heading": {
    "block_suffix": "\n",
    "bold":         true
  },
  "h1": {
    "block_prefix":      "\n",
    "block_suffix":      "\n\n",
    "bold":              true,
    "color":             "230",
    "background_color":  "62",
    "prefix":            "  ",
    "suffix":            "  "
  },
  "h2": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "bold":         true,
    "color":        "111",
    "prefix":       "── "
  },
  "h3": {
    "block_prefix": "\n",
    "block_suffix": "\n",
    "bold":         true,
    "color":        "183"
  },
  "h4": {
    "bold":  true,
    "color": "152"
  },
  "h5": {
    "bold":  true,
    "color": "246"
  },
  "h6": {
    "bold":  true,
    "color": "242"
  },
  "strikethrough": {
    "crossed_out": true,
    "color":       "244"
  },
  "emph": {
    "italic": true,
    "color":  "189"
  },
  "strong": {
    "bold":  true,
    "color": "255"
  },
  "hr": {
    "color":        "241",
    "block_prefix": "\n",
    "block_suffix": "\n"
  },
  "item": {
    "block_prefix": "• "
  },
  "enumeration": {
    "block_prefix": ". "
  },
  "task": {
    "ticked":   "✓ ",
    "unticked": "○ "
  },
  "link": {
    "color":     "80",
    "underline": true
  },
  "link_text": {
    "color": "75",
    "bold":  true
  },
  "image": {
    "color":     "212",
    "underline": true
  },
  "image_text": {
    "color": "212"
  },
  "code": {
    "prefix":           " ",
    "suffix":           " ",
    "color":            "222",
    "background_color": "237"
  },
  "code_block": {
    "color":  "#d0d0d0",
    "margin": 1,
    "chroma": {
      "text":                     { "color": "#d0d0d0" },
      "error":                    { "color": "#ff5f5f", "background_color": "#5f0000" },
      "comment":                  { "color": "#767676", "italic": true },
      "comment_preproc":          { "color": "#ffd75f" },
      "comment_preproc_file":     { "color": "#ffd75f" },
      "comment_single":           { "color": "#767676", "italic": true },
      "comment_multiline":        { "color": "#767676", "italic": true },
      "keyword":                  { "color": "#87afd7", "bold": true },
      "keyword_constant":         { "color": "#87afd7", "bold": true },
      "keyword_declaration":      { "color": "#87afd7", "bold": true },
      "keyword_namespace":        { "color": "#87afd7", "bold": true },
      "keyword_pseudo":           { "color": "#87afd7" },
      "keyword_reserved":         { "color": "#87afd7", "bold": true },
      "keyword_type":             { "color": "#87d7d7" },
      "operator":                 { "color": "#87afd7" },
      "operator_word":            { "color": "#87afd7", "bold": true },
      "punctuation":              { "color": "#d0d0d0" },
      "name":                     { "color": "#e4e4e4" },
      "name_attribute":           { "color": "#ffd75f" },
      "name_builtin":             { "color": "#afd787" },
      "name_builtin_pseudo":      { "color": "#afd787" },
      "name_class":               { "color": "#87d7d7", "bold": true },
      "name_constant":            { "color": "#ff87d7" },
      "name_decorator":           { "color": "#d7afff" },
      "name_entity":              { "color": "#ffd75f" },
      "name_exception":           { "color": "#ff5f5f" },
      "name_function":            { "color": "#87d7d7" },
      "name_function_magic":      { "color": "#87d7d7" },
      "name_label":               { "color": "#ffd75f" },
      "name_namespace":           { "color": "#87d7d7" },
      "name_other":               { "color": "#e4e4e4" },
      "name_tag":                 { "color": "#87afd7" },
      "name_variable":            { "color": "#e4e4e4" },
      "name_variable_class":      { "color": "#87d7d7" },
      "name_variable_global":     { "color": "#ff87d7" },
      "name_variable_instance":   { "color": "#e4e4e4" },
      "name_variable_magic":      { "color": "#ff87d7" },
      "literal_number":           { "color": "#ff87d7" },
      "literal_number_bin":       { "color": "#ff87d7" },
      "literal_number_float":     { "color": "#ff87d7" },
      "literal_number_hex":       { "color": "#ff87d7" },
      "literal_number_integer":   { "color": "#ff87d7" },
      "literal_number_integer_long": { "color": "#ff87d7" },
      "literal_number_oct":       { "color": "#ff87d7" },
      "literal_string":           { "color": "#afd787" },
      "literal_string_affix":     { "color": "#afd787" },
      "literal_string_backtick":  { "color": "#afd787" },
      "literal_string_char":      { "color": "#afd787" },
      "literal_string_delimiter": { "color": "#afd787" },
      "literal_string_doc":       { "color": "#808080", "italic": true },
      "literal_string_double":    { "color": "#afd787" },
      "literal_string_escape":    { "color": "#ffd75f" },
      "literal_string_heredoc":   { "color": "#afd787" },
      "literal_string_interpol":  { "color": "#ff5f5f" },
      "literal_string_other":     { "color": "#afd787" },
      "literal_string_regex":     { "color": "#ffd75f" },
      "literal_string_single":    { "color": "#afd787" },
      "literal_string_symbol":    { "color": "#ffd75f" },
      "generic_deleted":          { "color": "#ff5f5f" },
      "generic_emph":             { "italic": true },
      "generic_heading":          { "color": "#87afff", "bold": true },
      "generic_inserted":         { "color": "#afd787" },
      "generic_output":           { "color": "#808080" },
      "generic_prompt":           { "color": "#87afd7", "bold": true },
      "generic_strong":           { "bold": true },
      "generic_subheading":       { "color": "#d7afff" },
      "generic_traceback":        { "color": "#ff5f5f" },
      "background":               { "background_color": "#1c1c1c" }
    }
  },
  "table": {
    "center_separator": "┼",
    "column_separator": "│",
    "row_separator":    "─"
  },
  "definition_list": {},
  "definition_term": {
    "bold":  true,
    "color": "111"
  },
  "definition_description": {
    "block_prefix": "  ",
    "color":        "252"
  },
  "html_block":  {},
  "html_inline": {}
}`
