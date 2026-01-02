// Skillrunner CLI entry point
//
// Skillrunner (sr) is a local-first AI workflow orchestration tool.
// It enables multi-phase AI workflows that prioritize local LLM providers
// (like Ollama) while seamlessly falling back to cloud providers when needed.
package main

import "github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands"

func main() {
	commands.Execute()
}
