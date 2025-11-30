package embedded

import (
	"embed"
	"io/fs"
)

//go:embed skills/*.yaml
var skillsFS embed.FS

// GetSkillsFS returns the embedded filesystem containing bundled skills.
// The filesystem is rooted at the 'skills' directory.
func GetSkillsFS() (fs.FS, error) {
	return fs.Sub(skillsFS, "skills")
}
