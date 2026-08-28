import { expect, test } from "@playwright/test"
import { formatDecimalAmount, formatMinorAmount } from "../src/lib/display"

test("amount formatting avoids floating-point arithmetic", () => {
  expect(formatMinorAmount(128250)).toBe("1282.5")
  expect(formatMinorAmount("900719925474099301")).toBe("9007199254740993.01")
  expect(formatMinorAmount(-1)).toBe("-0.01")
  expect(formatMinorAmount(1200, { useGrouping: true })).toBe("12")
  expect(formatMinorAmount("123456789012345", { useGrouping: true })).toBe("1,234,567,890,123.45")
  expect(formatMinorAmount(1200, { fractionDigits: 0 })).toBe("1200")
  expect(formatMinorAmount(1200, { fractionDigits: 21 })).toBeNull()
  expect(formatMinorAmount(9007199254740992)).toBeNull()

  expect(formatDecimalAmount("0012.500")).toBe("12.5")
  expect(formatDecimalAmount("9007199254740993.0100")).toBe("9007199254740993.01")
})
