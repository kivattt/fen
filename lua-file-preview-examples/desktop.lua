--[[
-- Works for fen v1.1.2 or above
-- File preview script for *.desktop files
--]]

-- https://specifications.freedesktop.org/desktop-entry-spec/desktop-entry-spec-latest.html

-- Customize the colors here:
local colorForGroupHeaders = "orange"

local colorForKeys = fen:ColorToString(fen:NewRGBColor(0, 255, 0)) -- Green
local attributesForKeys = "b" -- Bold

local colorForEquals = "blue"
local colorForValues = "default"

local State = {
	Key = 1,
	Value = 2,
}

local y = 0
for line in io.lines(fen.selectedFile) do
	local state = State.Key
	local seenEqualsSign = false

	-- Group header
	if line:sub(1,1) == '[' then
		fen:PrintSimple("["..colorForGroupHeaders.."]"..line, 0, y)
		goto continueLine
	end

	for i = 1, #line do
		local char = line:sub(i,i)

		if char == '=' and not seenEqualsSign then
			seenEqualsSign = true
			state = State.Value
			fen:PrintSimple("["..colorForEquals.."]"..char, i-1, y)
			goto continue
		end

		local foreground = "default"
		local attributes = ""
		if state == State.Key then
			foreground = colorForKeys
			attributes = attributesForKeys
		elseif state == State.Value then
			foreground = colorForValues
		end

		fen:PrintSimple("["..foreground.."::"..attributes.."]"..char, i-1, y)
		::continue::
	end

	::continueLine::
	y = y + 1
	if y >= fen.Height then
		break
	end
end
