-- This script opens a new editor consecutively for every selected file

local editor = os.getenv("EDITOR")
if fen.RuntimeOS == "windows" then
	editor = "notepad"
end

for i = 1, #fen.SelectedFiles do
	os.execute(editor.." "..fen.SelectedFiles[i])
end
