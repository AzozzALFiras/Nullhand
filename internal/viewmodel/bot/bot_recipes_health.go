package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
	reciperepo "github.com/AzozzALFiras/Nullhand/internal/repository/recipe"
	ocrsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/ocr"
	permsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/permissions"
	recipesvc "github.com/AzozzALFiras/Nullhand/internal/service/recipe"
	transcribesvc "github.com/AzozzALFiras/Nullhand/internal/service/transcribe"
)

// handleRecipesCommand processes /recipes [subcommand] [args...].
//
// Forms:
//
//	/recipes                       — list every recipe (built-in + user)
//	/recipes <name>                — show steps of one recipe
//	/recipes show <name>           — same as above
//	/recipes run <name> [k=v ...]  — execute a recipe with inline params
//	/recipes delete <name>         — remove a user-saved recipe
//	/recipes rename <old> <new>    — rename a user recipe
func (vm *ViewModel) handleRecipesCommand(chatID, userID int64, args []string) {
	svc := vm.agent.Recipes()
	if svc == nil {
		vm.send(chatID, "❌ Recipe service is not available.")
		return
	}

	if len(args) == 0 {
		vm.send(chatID, formatRecipeList(svc, configBuiltinNames(svc)))
		return
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "show", "view":
		if len(args) < 2 {
			vm.send(chatID, "Usage: /recipes show <name>")
			return
		}
		vm.send(chatID, formatRecipeDetails(svc, args[1]))

	case "run", "exec":
		if len(args) < 2 {
			vm.send(chatID, "Usage: /recipes run <name> [key=value ...]")
			return
		}
		name := args[1]
		params := parseInlineParams(args[2:])
		vm.runRecipeAndReply(chatID, userID, name, params, false)

	case "preview":
		if len(args) < 2 {
			vm.send(chatID, "Usage: /recipes preview <name> [key=value ...]")
			return
		}
		name := args[1]
		params := parseInlineParams(args[2:])
		vm.runRecipeAndReply(chatID, userID, name, params, true)

	case "delete", "remove", "rm":
		if len(args) < 2 {
			vm.send(chatID, "Usage: /recipes delete <name>")
			return
		}
		vm.deleteRecipe(chatID, userID, args[1])

	case "rename", "mv":
		if len(args) < 3 {
			vm.send(chatID, "Usage: /recipes rename <oldName> <newName>")
			return
		}
		vm.renameRecipe(chatID, userID, args[1], args[2])

	case "export":
		vm.exportRecipes(chatID, userID, svc)

	case "import":
		vm.armRecipeImport(chatID, userID)

	default:
		// Treat the first arg as a recipe name to show.
		vm.send(chatID, formatRecipeDetails(svc, args[0]))
	}
}

// exportRecipes serialises every user-defined recipe (built-ins are skipped
// since they ship with the binary) and uploads it as a Telegram document.
// The on-disk file is never touched — we serialise from the in-memory set so
// any unsaved edits made via "save this as recipe X" are included.
func (vm *ViewModel) exportRecipes(chatID, userID int64, svc *recipesvc.Service) {
	builtins := configBuiltinNames(svc)
	user := make(map[string]recipemodel.Recipe)
	for name, r := range svc.All() {
		if builtins[name] {
			continue
		}
		user[name] = r
	}
	if len(user) == 0 {
		vm.send(chatID, "ℹ️ No user recipes to export. Save one first with \"save this as recipe X: ...\"")
		return
	}

	file := recipemodel.File{Version: 1, Recipes: user}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Export failed: %v", err))
		return
	}

	filename := fmt.Sprintf("nullhand-recipes-%s.json", time.Now().Format("20060102-150405"))
	if err := vm.tg.SendDocument(chatID, data, filename); err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Failed to upload: %v", err))
		return
	}
	vm.auditLog(userID, "recipes_export", fmt.Sprintf(`count=%d`, len(user)))
}

// armRecipeImport puts the chat into "expect a recipes JSON" state. The next
// document upload from this chat is consumed by handleRecipeImport instead of
// going through the normal file-receive flow.
func (vm *ViewModel) armRecipeImport(chatID, userID int64) {
	vm.recipeImportMu.Lock()
	if vm.recipeImportArmed == nil {
		vm.recipeImportArmed = make(map[int64]bool)
	}
	vm.recipeImportArmed[chatID] = true
	vm.recipeImportMu.Unlock()
	vm.auditLog(userID, "recipes_import_arm")
	vm.send(chatID, "📥 Send the recipes JSON file now (or any other command to cancel).")
}

// isRecipeImportArmed checks and clears the import-armed flag for chatID.
// Returns true if the next document should be treated as a recipe import.
func (vm *ViewModel) isRecipeImportArmed(chatID int64) bool {
	vm.recipeImportMu.Lock()
	defer vm.recipeImportMu.Unlock()
	if vm.recipeImportArmed == nil {
		return false
	}
	armed := vm.recipeImportArmed[chatID]
	delete(vm.recipeImportArmed, chatID)
	return armed
}

// handleRecipeImport parses an uploaded JSON file and merges the recipes into
// the in-memory set, persisting them to ~/.nullhand/recipes.json. Built-in
// recipes are protected: any incoming entry whose name collides with a
// built-in is skipped with a warning, since overwriting it would change
// behaviour the user did not author.
func (vm *ViewModel) handleRecipeImport(chatID, userID int64, data []byte, filename string) {
	svc := vm.agent.Recipes()
	if svc == nil {
		vm.send(chatID, "❌ Recipe service unavailable.")
		return
	}

	var file recipemodel.File
	if err := json.Unmarshal(data, &file); err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Invalid JSON in %q: %v", filename, err))
		return
	}
	if len(file.Recipes) == 0 {
		vm.send(chatID, "ℹ️ File contains no recipes.")
		return
	}

	builtins := configBuiltinNames(svc)
	added, replaced, skipped := 0, 0, 0
	for name, r := range file.Recipes {
		if builtins[name] {
			skipped++
			continue
		}
		if _, existed := svc.Get(name); existed {
			replaced++
		} else {
			added++
		}
		svc.Set(name, r)
	}

	// Persist user recipes only.
	user := make(map[string]recipemodel.Recipe)
	for name, r := range svc.All() {
		if builtins[name] {
			continue
		}
		user[name] = r
	}
	if err := reciperepo.Save(user); err != nil {
		vm.send(chatID, fmt.Sprintf("⚠️ Imported in memory but persistence failed: %v", err))
		return
	}

	vm.auditLog(userID, "recipes_import",
		fmt.Sprintf(`added=%d replaced=%d skipped=%d`, added, replaced, skipped))

	msg := fmt.Sprintf("✅ Imported recipes: %d added, %d replaced", added, replaced)
	if skipped > 0 {
		msg += fmt.Sprintf(", %d skipped (built-in name collision)", skipped)
	}
	vm.send(chatID, msg)
}

// formatRecipeList builds a human-readable index of all recipes, sorted by
// name. Built-in recipes are marked separately from user-defined ones.
func formatRecipeList(svc *recipesvc.Service, builtins map[string]bool) string {
	all := svc.List()
	if len(all) == 0 {
		return "No recipes available."
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 %d recipes available:\n\n", len(all)))

	var userRecipes []recipemodel.Recipe
	var builtinRecipes []recipemodel.Recipe
	for _, r := range all {
		if builtins[r.Name] {
			builtinRecipes = append(builtinRecipes, r)
		} else {
			userRecipes = append(userRecipes, r)
		}
	}

	if len(userRecipes) > 0 {
		sb.WriteString("⭐ Your recipes:\n")
		for _, r := range userRecipes {
			sb.WriteString(formatRecipeLine(r))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("📦 Built-in:\n")
	for _, r := range builtinRecipes {
		sb.WriteString(formatRecipeLine(r))
	}
	sb.WriteString("\nUse /recipes show <name> to see steps, /recipes run <name> to execute.")
	return sb.String()
}

func formatRecipeLine(r recipemodel.Recipe) string {
	desc := r.Description
	if desc == "" {
		desc = "(no description)"
	}
	if len(r.Parameters) > 0 {
		return fmt.Sprintf("• %s [params: %s] — %s\n", r.Name, strings.Join(r.Parameters, ", "), desc)
	}
	return fmt.Sprintf("• %s — %s\n", r.Name, desc)
}

// formatRecipeDetails returns a verbose view of one recipe's steps.
func formatRecipeDetails(svc *recipesvc.Service, name string) string {
	r, ok := svc.Get(name)
	if !ok {
		return fmt.Sprintf("❌ Recipe %q not found. Use /recipes to list available ones.", name)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📖 Recipe: %s\n", r.Name))
	if r.Description != "" {
		sb.WriteString("Description: " + r.Description + "\n")
	}
	if len(r.Parameters) > 0 {
		sb.WriteString("Parameters: " + strings.Join(r.Parameters, ", ") + "\n")
	}
	sb.WriteString(fmt.Sprintf("Steps (%d):\n", len(r.Steps)))
	for i, s := range r.Steps {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, describeRecipeStep(s)))
	}
	return sb.String()
}

// describeRecipeStep mirrors the formatter in recipe_service.go but lives here
// so we don't import internal helpers across packages.
func describeRecipeStep(s recipemodel.Step) string {
	switch s.Kind {
	case recipemodel.StepOpenApp:
		return fmt.Sprintf("open_app(%q)", s.AppName)
	case recipemodel.StepPressKey:
		return fmt.Sprintf("press_key(%q)", s.Key)
	case recipemodel.StepTypeText:
		return fmt.Sprintf("type_text(%q)", truncateStr(s.Text, 50))
	case recipemodel.StepPalette:
		return fmt.Sprintf("palette(%q, %q)", s.Shortcut, s.Command)
	case recipemodel.StepSleepMs:
		return fmt.Sprintf("sleep(%dms)", s.Ms)
	case recipemodel.StepFocusField:
		return fmt.Sprintf("focus_field(%q)", s.Label)
	case recipemodel.StepWaitForWindow:
		return fmt.Sprintf("wait_for_window(%q, %dms)", s.Text, s.Ms)
	case recipemodel.StepWaitForText:
		return fmt.Sprintf("wait_for_text(%q, %dms)", truncateStr(s.Text, 40), s.Ms)
	case recipemodel.StepWaitForElement:
		return fmt.Sprintf("wait_for_element(%q, %dms)", s.Label, s.Ms)
	case recipemodel.StepClickText:
		return fmt.Sprintf("click_text(%q)", truncateStr(s.Text, 40))
	case recipemodel.StepClickFuzzy:
		return fmt.Sprintf("click_fuzzy(%q)", s.Label)
	case recipemodel.StepClearField:
		return "clear_field()"
	default:
		return string(s.Kind)
	}
}

// runRecipeAndReply executes a recipe (or dry-runs it) and sends the result
// back to the user. Used by /recipes run|preview.
func (vm *ViewModel) runRecipeAndReply(chatID, userID int64, name string, params map[string]string, dryRun bool) {
	svc := vm.agent.Recipes()
	plan, err := svc.Run(name, params, dryRun)
	if err != nil {
		vm.send(chatID, fmt.Sprintf("❌ %v\n%s", err, plan))
		return
	}
	if dryRun {
		vm.auditLog(userID, "recipe_preview", fmt.Sprintf(`name=%q`, name))
		vm.send(chatID, fmt.Sprintf("🔎 Dry-run of %s:\n%s", name, plan))
		return
	}
	vm.auditLog(userID, "recipe_run", fmt.Sprintf(`name=%q`, name))
	vm.send(chatID, fmt.Sprintf("✅ Ran %s:\n%s", name, plan))
}

// deleteRecipe removes a user-defined recipe. Built-in recipes can't be
// deleted (they're hard-coded in defaults.go); attempting that returns a
// clear error.
func (vm *ViewModel) deleteRecipe(chatID, userID int64, name string) {
	svc := vm.agent.Recipes()
	if _, ok := svc.Get(name); !ok {
		vm.send(chatID, fmt.Sprintf("❌ Recipe %q does not exist.", name))
		return
	}
	if configBuiltinNames(svc)[name] {
		vm.send(chatID, fmt.Sprintf("❌ Recipe %q is a built-in default and can't be deleted. You can shadow it by saving a new recipe with the same name.", name))
		return
	}
	svc.Delete(name)
	if err := persistUserRecipes(svc); err != nil {
		vm.send(chatID, fmt.Sprintf("⚠️ Removed in memory but disk write failed: %v", err))
		return
	}
	vm.auditLog(userID, "recipe_delete", fmt.Sprintf(`name=%q`, name))
	vm.send(chatID, fmt.Sprintf("✅ Deleted recipe %q.", name))
}

// renameRecipe renames a user-defined recipe in the in-memory map and on disk.
func (vm *ViewModel) renameRecipe(chatID, userID int64, oldName, newName string) {
	svc := vm.agent.Recipes()
	r, ok := svc.Get(oldName)
	if !ok {
		vm.send(chatID, fmt.Sprintf("❌ Recipe %q does not exist.", oldName))
		return
	}
	if configBuiltinNames(svc)[oldName] {
		vm.send(chatID, fmt.Sprintf("❌ Recipe %q is a built-in default and can't be renamed. Save a copy under the new name instead.", oldName))
		return
	}
	if _, exists := svc.Get(newName); exists {
		vm.send(chatID, fmt.Sprintf("❌ Recipe %q already exists. Pick a different new name.", newName))
		return
	}
	r.Name = newName
	svc.Set(newName, r)
	svc.Delete(oldName)
	if err := persistUserRecipes(svc); err != nil {
		vm.send(chatID, fmt.Sprintf("⚠️ Renamed in memory but disk write failed: %v", err))
		return
	}
	vm.auditLog(userID, "recipe_rename", fmt.Sprintf(`from=%q to=%q`, oldName, newName))
	vm.send(chatID, fmt.Sprintf("✅ Renamed %q → %q.", oldName, newName))
}

// configBuiltinNames returns a set of recipe names that ship with Nullhand
// (built-in defaults). Used to decide which recipes a user is allowed to
// delete or rename.
func configBuiltinNames(svc *recipesvc.Service) map[string]bool {
	out := map[string]bool{}
	for name := range recipesvc.Defaults() {
		out[name] = true
	}
	return out
}

// persistUserRecipes writes only USER-DEFINED recipes to ~/.nullhand/recipes.json.
// Built-in defaults are not persisted because they ship in code.
func persistUserRecipes(svc *recipesvc.Service) error {
	builtins := configBuiltinNames(svc)
	user := map[string]recipemodel.Recipe{}
	for name, r := range svc.All() {
		if builtins[name] {
			continue
		}
		user[name] = r
	}
	return reciperepo.Save(user)
}

// parseInlineParams turns ["contact=Azozz", "message=hello world"] into
// {"contact":"Azozz", "message":"hello world"}. Values containing spaces
// must be supplied as a single Telegram-quoted argument by the user — the
// router already handles this.
func parseInlineParams(args []string) map[string]string {
	out := map[string]string{}
	for _, a := range args {
		idx := strings.Index(a, "=")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(a[:idx])
		v := strings.TrimSpace(a[idx+1:])
		v = strings.Trim(v, `"'`)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

// handleHealthCommand prints a comprehensive system / dependency status
// report. Useful for debugging "why doesn't X work?" without digging into
// logs.
func (vm *ViewModel) handleHealthCommand(chatID, userID int64) {
	vm.auditLog(userID, "health")
	vm.send(chatID, vm.formatHealth())
}

// formatHealth assembles the /health report string.
func (vm *ViewModel) formatHealth() string {
	var sb strings.Builder
	sb.WriteString("🩺 Nullhand health report\n\n")

	// Platform
	sb.WriteString(fmt.Sprintf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	// AI provider
	provider := vm.cfg.AIProvider
	if provider == "" {
		provider = "(unset)"
	}
	sb.WriteString(fmt.Sprintf("AI provider: %s\n", provider))
	if vm.cfg.AIModel != "" {
		sb.WriteString(fmt.Sprintf("AI model: %s\n", vm.cfg.AIModel))
	}

	// OCR languages
	sb.WriteString(fmt.Sprintf("OCR languages: %s\n", ocrsvc.Languages()))

	// Voice transcription (whisper.cpp + ffmpeg)
	if ok, label := transcribesvc.IsAvailable(); ok {
		sb.WriteString(fmt.Sprintf("Voice transcription: ✅ %s + ffmpeg\n", label))
	} else {
		sb.WriteString(fmt.Sprintf("Voice transcription: ❌ %s — voice notes will fail\n", label))
	}

	// Permissions / capabilities
	status := permsvc.Check()
	sb.WriteString(fmt.Sprintf("Screen Recording: %s\n", okMark(status.ScreenRecording)))
	sb.WriteString(fmt.Sprintf("Accessibility:    %s\n", okMark(status.Accessibility)))

	// Scheduler
	if tasks := vm.scheduler.List(); len(tasks) > 0 {
		sb.WriteString(fmt.Sprintf("\nScheduled tasks (%d):\n", len(tasks)))
		for _, t := range tasks {
			sb.WriteString(fmt.Sprintf("  • %s — %s — %s\n", t.ID, t.Label, formatScheduleHuman(t)))
		}
	} else {
		sb.WriteString("\nScheduled tasks: none\n")
	}

	// Recipes
	if r := vm.agent.Recipes(); r != nil {
		all := r.List()
		builtins := configBuiltinNames(r)
		userN := 0
		for _, rec := range all {
			if !builtins[rec.Name] {
				userN++
			}
		}
		sb.WriteString(fmt.Sprintf("\nRecipes: %d total (%d built-in, %d user-defined)\n", len(all), len(all)-userN, userN))
	}

	// Storage: audit log writability + free disk space on $HOME/.nullhand
	if home, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(home, ".nullhand")
		sb.WriteString(fmt.Sprintf("\nStorage dir: %s\n", dir))
		sb.WriteString(fmt.Sprintf("Audit log writable: %s\n", okMark(isDirWritable(dir))))
		if free, ok := freeDiskBytes(dir); ok {
			sb.WriteString(fmt.Sprintf("Free disk space: %s\n", humanBytes(free)))
		}
	}

	// Allowed users (legacy single ID + the list)
	allowed := vm.guard.AllowedUserIDs()
	sort.Slice(allowed, func(i, j int) bool { return allowed[i] < allowed[j] })
	if len(allowed) <= 1 {
		sb.WriteString(fmt.Sprintf("\nAllowed Telegram user: %d\n", vm.cfg.AllowedUserID))
	} else {
		sb.WriteString(fmt.Sprintf("\nAllowed Telegram users (%d):\n", len(allowed)))
		for _, id := range allowed {
			sb.WriteString(fmt.Sprintf("  • %d\n", id))
		}
	}
	sb.WriteString(fmt.Sprintf("Session unlocked: %v\n", vm.otp.IsUnlocked()))

	return sb.String()
}

// isDirWritable returns true if dir exists and the process can create files
// in it. Uses a probe file so we never have to interpret stat permission bits
// across platforms / ACLs.
func isDirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return false
	}
	probe, err := os.CreateTemp(dir, ".healthprobe-*")
	if err != nil {
		return false
	}
	probePath := probe.Name()
	_ = probe.Close()
	_ = os.Remove(probePath)
	return true
}

// freeDiskBytes returns the number of free bytes on the filesystem hosting
// `path`. The second return is false when the call is unsupported or fails.
func freeDiskBytes(path string) (uint64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, false
	}
	return uint64(st.Bavail) * uint64(st.Bsize), true
}

func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func okMark(b bool) string {
	if b {
		return "✅ ok"
	}
	return "❌ missing — see /diag for details"
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

