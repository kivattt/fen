--[[
-- File preview for go.mod files in Golang projects
--]]
--
local function trimLeftAndRightSpaces(s)
   return (s:gsub("^%s*(.-)%s*$", "%1"))
end

local y = 0
local indirect = "// indirect"

local module = "module"
local go = "go"
local replace = "replace"
local require = "require"

local replacedDependencies = {}

for line in io.lines(fen.SelectedFile) do
	local style = ""

	if line:sub(1,2) == "//" then
		style = "[blue::d]"
	elseif line:sub(1, #replace) == replace then
		local separator = line:find("=>")
		if separator ~= nil then
			local replacing = line:sub(#replace+2, separator - 1)
			local replacingWith = line:sub(separator + 2)
			fen:PrintSimple("[yellow]replace [red]"..replacing.."[yellow]=>[default]"..replacingWith, 0, y)

			replacedDependencies[trimLeftAndRightSpaces(replacing)] = true
			goto continueLine
		end

		style = "[yellow]"
	elseif line:sub(-#indirect) == indirect then
		style = "[gray]"
	elseif line:sub(1, #module) == module then
		style = "[yellow]"
	elseif line:sub(1, #go) == go then
		style = "[yellow]"
	elseif line:sub(1, #require) == require then
		style = "[yellow]"
	elseif line == ")" then
		style = "[yellow]"
	else
		local words = {}
		for word in line:gmatch("%S+") do table.insert(words, word) end
		if replacedDependencies[words[#words-1]] ~= nil then
			style = "[red]"
		end
	end

	fen:PrintSimple(style..fen:Escape(line), 0, y)

	::continueLine::
	y = y + 1
	if y >= fen.Height then
		break
	end
end
