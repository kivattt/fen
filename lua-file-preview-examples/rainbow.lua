-- https://stackoverflow.com/a/71365991
local function HSV2RGB (h, s, v)
    local k1 = v*(1-s)
    local k2 = v - k1
    local r = math.min (math.max (3*math.abs (((h      )/180)%2-1)-1, 0), 1)
    local g = math.min (math.max (3*math.abs (((h  -120)/180)%2-1)-1, 0), 1)
    local b = math.min (math.max (3*math.abs (((h  +120)/180)%2-1)-1, 0), 1)
    return k1 + k2 * r * 255, k1 + k2 * g * 255, k1 + k2 * b * 255
end


local y = 0
local hue = 0
for line in io.lines(fen.SelectedFile) do
	fen:Print(fen:Escape(line), 0, y, fen.Width, 0, fen:NewRGBColor(HSV2RGB(hue, 1, 1)))
	hue = hue + 10

	y = y + 1
	if y >= fen.Height then
		break
	end
end
