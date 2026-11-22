import { afterEach } from "vitest";
import { cleanupApp } from "./test-utils";

afterEach(async () => {
  await cleanupApp();
});
