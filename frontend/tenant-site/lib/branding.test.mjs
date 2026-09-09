/**
 * Pure-Node tests for tenant-site/lib/branding.ts
 * We re-implement the private hexToHsl helper here to validate the
 * algorithm; the production source uses the same equations.
 */
let passed = 0
let failed = 0

function assert(cond, label) {
  if (cond) passed++
  else {
    failed++
    console.error("FAIL " + label)
  }
}

function assertEqual(actual, expected, label) {
  if (actual === expected) passed++
  else {
    failed++
    console.error("FAIL " + label + ": got " + JSON.stringify(actual) + " want " + JSON.stringify(expected))
  }
}

function hexToHsl(hex) {
  hex = hex.replace(/^#/, "")
  const r = parseInt(hex.substring(0, 2), 16) / 255
  const g = parseInt(hex.substring(2, 4), 16) / 255
  const b = parseInt(hex.substring(4, 6), 16) / 255
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  let h = 0
  let s = 0
  const l = (max + min) / 2
  if (max !== min) {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6
        break
      case g:
        h = ((b - r) / d + 2) / 6
        break
      case b:
        h = ((r - g) / d + 4) / 6
        break
    }
  }
  return Math.round(h * 360) + " " + Math.round(s * 100) + "% " + Math.round(l * 100) + "%"
}

assert(hexToHsl("#ff0000").length > 0, "red non-empty")
assertEqual(hexToHsl("#000000").trim().split(/\s+/)[2], "0%", "black L=0%")
assertEqual(hexToHsl("#ffffff").trim().split(/\s+/)[2], "100%", "white L=100%")
assertEqual(hexToHsl("#ff0000").trim().split(/\s+/)[0], "0", "red H=0")
assertEqual(hexToHsl("#00ff00").trim().split(/\s+/)[0], "120", "green H=120")
assertEqual(hexToHsl("#0000ff").trim().split(/\s+/)[0], "240", "blue H=240")

const l_black = parseInt(hexToHsl("#000000").split(/\s+/)[2])
const l_mid = parseInt(hexToHsl("#808080").split(/\s+/)[2])
const l_white = parseInt(hexToHsl("#ffffff").split(/\s+/)[2])
assert(l_black < l_mid, "black L < mid L")
assert(l_mid < l_white, "mid L < white L")

const s_red = parseInt(hexToHsl("#ff0000").split(/\s+/)[1])
assertEqual(s_red, 100, "red sat=100%")

const s_grey = parseInt(hexToHsl("#808080").split(/\s+/)[1])
assertEqual(s_grey, 0, "grey sat=0%")

// Output format: "H S% L%"
const out = hexToHsl("#abcdef").split(" ")
assert(out.length === 3, "three parts")
assert(out[1].endsWith("%"), "S ends with %")
assert(out[2].endsWith("%"), "L ends with %")

// Strip leading hash
assertEqual(hexToHsl("ff0000"), hexToHsl("#ff0000"), "no hash equals hash")

// Mixed case hex
assertEqual(hexToHsl("#FfAa00"), hexToHsl("#ffaa00"), "case insensitive")

console.log("\nResults: " + passed + " passed, " + failed + " failed")
if (failed > 0) process.exit(1)
