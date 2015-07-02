{{if gt (len .Fields) 0}}
| Field | Type | Required? | Description |
| :-- | :-- | :-- | :--------- |
{{range .Fields}}| `{{.Name}}` | {{linkType .TypeName}} | {{if .Optional}}*optional*{{else}}required{{end}} | {{.Comments}} |
{{end}}
{{else}}
This packet has no fields.
{{end}}
