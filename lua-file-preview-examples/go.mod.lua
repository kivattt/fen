--[[
-- File preview for go.mod files in Golang projects
--]]

-- Customize the colors here:
local colorForDirectives = "[orange]"
local colorForReplacedDependencies = "[red]"
local colorForComments = "[teal]"
local colorForIndirectDependencies = "[gray]"

local function trimLeftAndRightSpaces(s)
   return (s:gsub("^%s*(.-)%s*$", "%1"))
end

local y = 0

local directives = {
	"module",
	"go",
	"toolchain",
	"godebug",
	"require",
	"replace",
	"exclude",
	"retract",
}

local function is_directive(line)
	for _, v in ipairs(directives) do
		if line:sub(1, #v) == v then
			return true
		end
	end
	return false
end

local indirect = "// indirect"
local replace = "replace"

local replacedDependencies = {}

for line in io.lines(fen.SelectedFile) do
	local style = ""

	if line:sub(1,2) == "//" then
		style = colorForComments
	elseif line:sub(1, #replace) == replace then
		local separator = line:find("=>")
		if separator ~= nil then
			local replacing = line:sub(#replace+2, separator - 1)
			local replacingWith = line:sub(separator + 2)
			fen:PrintSimple(colorForDirectives.."replace "..colorForReplacedDependencies..replacing..colorForDirectives.."=>[default]"..replacingWith, 0, y)

			replacedDependencies[trimLeftAndRightSpaces(replacing)] = true
			goto continueLine
		end

		style = colorForDirectives
	elseif line:sub(-#indirect) == indirect then
		style = colorForIndirectDependencies
	elseif line == ")" or is_directive(line) then
		style = colorForDirectives
	else
		local words = {}
		for word in line:gmatch("%S+") do table.insert(words, word) end
		if replacedDependencies[words[#words-1]] ~= nil then
			style = colorForReplacedDependencies
		end
	end

	fen:PrintSimple(style..fen:Escape(line), 0, y)

	::continueLine::
	y = y + 1
	if y >= fen.Height then
		break
	end
end
