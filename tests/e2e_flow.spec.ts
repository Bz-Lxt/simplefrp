import { test, expect } from "@playwright/test";

const visitor = process.env.VISITOR_URL || "http://127.0.0.1:42817";

test("visitor portal shows intranet node", async ({ page }) => {
  await page.goto(visitor);
  await expect(page.getByRole("heading", { name: "内网节点观察窗" })).toBeVisible();
  await expect(page.getByText("SIMPLEFRP")).toBeVisible();
  await page.getByRole("button", { name: "重新探测身份" }).click();
  await expect(page.getByText("intranet-alpha-7").first()).toBeVisible({ timeout: 10_000 });
  await expect(page.getByText("THROUGH").first()).toBeVisible();
});
