-- All available options, default values
fen.ui_borders = false -- When set to true it will slow down fen a little
fen.no_write = false
fen.print_path_on_open = false
fen.mouse = true
fen.hidden_files = false -- Show hidden files
fen.folders_first = true
fen.split_home_end = false -- Only applies when fen.folders_first = true, splits Home/End keys behaviour over files and folders
fen.terminal_title = true -- Only applies to Linux, sets the terminal title to "fen <version>"
fen.show_hostname = true -- Does not apply to Windows, shows username@hostname in the top left
fen.show_help_text = true
fen.sort_by = "alphabetical" -- "fen -h" for valid values
fen.sort_reverse = false
fen.file_event_interval_ms = 300 -- How often to update the screen on file events (and job count updates), if set to 0, it updates on every event
fen.always_show_info_numbers = false -- Shows the blue, green and yellow numbers in the bottom right even when they are 0
fen.scroll_speed = 2 -- When scrolling faster than 30ms per scroll, scroll this many entries
fen.git_status = false -- EXPERIMENTAL: When true, unstaged/untracked files in local git repositories are shown in red
fen.preview_safety_blocklist = true -- Prevents common sensitive file types from being previewed

-- Everything below this line is non-default examples

-- The fen version string
print(fen.version) -- Something like "v1.6.6"

-- The current operating system
print(fen.runtime_os) -- "linux", "darwin" (for macOS), "freebsd", "windows"

-- The OS-specific config path OR parent directory of `--config` argument
print(fen.config_path) -- Something like "/home/YOUR_USER/.config/fen/" (always ends in a slash)

-- The OS-specific home folder (nil if not found), details: https://pkg.go.dev/os#UserHomeDir
print(fen.home_path) -- Something like "/home/YOUR_USER/" (always ends in a slash)

-- When pressing a number key (0-9), go to the specified folder or file path
-- It can also be a relative path which can be used in any folder
-- This is a list with no more than 10 elements
fen.bookmarks = {
	[1] = fen.home_path,
	[2] = fen.config_path .. "config.lua",
	[3] = fen.home_path .. "Documents",
	[4] = fen.home_path .. "Desktop",
	[5] = fen.home_path .. "Downloads",
	[6] = fen.home_path .. "Music",
	[7] = fen.home_path .. "Pictures",
	[8] = fen.home_path .. "Videos",
	[9] = fen.home_path .. "Users",
	[10] = "/", -- This is used when pressing '0',
}

-- You can use fen.runtime_os to let your config have specific behaviour on different operating systems
local textEditor = os.getenv("EDITOR")
if fen.runtime_os == "windows" then
	textEditor = "notepad"
end

-- All keys except "script" are lists, but if you only need 1 element you can omit the curly braces

-- When you press right arrow, L or Enter, you open a file.
-- You can customize what programs to use for different file types
fen.open = {
	{
		program = {textEditor},
		match = {"*"}, -- A "*" matches any file, details: https://pkg.go.dev/path/filepath#Match
		do_not_match = {"*.exe"} -- You can exclude certain file types
	},
	{
		-- You can use Lua scripts to open files
		script = fen.config_path.."open.lua",
		match = {"*"}
	}
}

-- In both fen.open and fen.preview, the program to use is determined by the first match.
-- Entries lower down will only be used if the ones above did not match.
--
-- Values in "program" do not expand tildes like "~/some/file.sh"
-- If you want to use a shell script, it has to have a shebang or you need to explicitly invoke the appropriate shell like "bash /some/file.sh"
fen.preview = {
	{
		-- If the first command exits with a non-zero exit code, the next one in the list will be ran
		program = {"head -n 100", "cat"}, -- You can use ' ' for command-line arguments
		match = {"*"},
		do_not_match = {"*.txt", "*.md"} -- When you've selected a .txt or .md file, the Lua script below will preview it
	},
	{
		-- You can use Lua scripts to preview files
		-- https://github.com/kivattt/fen/blob/main/lua-file-preview-examples/rainbow.lua
		script = fen.config_path.."rainbow.lua",
		match = {"*"},
	}
}
