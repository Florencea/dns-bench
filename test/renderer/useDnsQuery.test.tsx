import { act, StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import type { Mock } from "vitest";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";
import { useDnsQuery } from "../../src/renderer/hooks/useDnsQuery";
import type { ModeT } from "../../src/shared/types";

type QueryFn = (mode: ModeT, name: string) => Promise<string[]>;

function createDeferred<T>(): {
  promise: Promise<T>;
  resolve: (value: T) => void;
} {
  let resolveRef: ((value: T) => void) | undefined;
  const promise = new Promise<T>((res) => {
    resolveRef = res;
  });
  return {
    promise,
    resolve: (value: T) => {
      resolveRef?.(value);
    },
  };
}

const HookHarness = () => {
  const { ipv4Data, ipv6Data, isLoading, runQuery, reset } = useDnsQuery();

  return (
    <div>
      <button
        type="button"
        onClick={() => {
          void runQuery("dns", "first.example.com");
        }}
      >
        Query First
      </button>
      <button
        type="button"
        onClick={() => {
          void runQuery("doh", "second.example.com");
        }}
      >
        Query Second
      </button>
      <button type="button" onClick={reset}>
        Reset Query
      </button>
      <output data-testid="loading">{isLoading ? "loading" : "idle"}</output>
      <output data-testid="ipv4">{ipv4Data?.join("|") ?? "empty"}</output>
      <output data-testid="ipv6">{ipv6Data?.join("|") ?? "empty"}</output>
    </div>
  );
};

describe("useDnsQuery", () => {
  let root: Root | null = null;
  let queryIpv4Mock: Mock<QueryFn>;
  let queryIpv6Mock: Mock<QueryFn>;

  async function mountHarness(): Promise<void> {
    let container = document.getElementById("hook-root");
    if (container === null) {
      container = document.createElement("div");
      container.id = "hook-root";
      document.body.appendChild(container);
    }
    root = createRoot(container);
    await act(async () => {
      await Promise.resolve();
      root?.render(
        <StrictMode>
          <HookHarness />
        </StrictMode>,
      );
    });
  }

  beforeEach(() => {
    queryIpv4Mock = vi.fn<QueryFn>().mockResolvedValue([]);
    queryIpv6Mock = vi.fn<QueryFn>().mockResolvedValue([]);

    window.api = {
      servers: [],
      queryIpv4: queryIpv4Mock,
      queryIpv6: queryIpv6Mock,
      getLocale: vi.fn(async () => await Promise.resolve("en-US" as const)),
      setLocale: vi.fn(async () => {
        await Promise.resolve();
      }),
      onLocaleChange: vi.fn(() => () => undefined),
    };
  });

  afterEach(async () => {
    if (root !== null) {
      await act(async () => {
        await Promise.resolve();
        root?.unmount();
      });
      root = null;
    }
    document.getElementById("hook-root")?.remove();
  });

  it("ignores stale responses when reset is called mid-flight", async () => {
    const deferredIpv4 = createDeferred<string[]>();
    queryIpv4Mock.mockImplementationOnce(
      async () => await deferredIpv4.promise,
    );

    await mountHarness();

    await page.getByRole("button", { name: "Query First" }).click();
    await expect
      .element(page.getByTestId("loading"))
      .toHaveTextContent("loading");

    await page.getByRole("button", { name: "Reset Query" }).click();
    await expect.element(page.getByTestId("loading")).toHaveTextContent("idle");

    deferredIpv4.resolve(["192.0.2.99"]);
    await act(async () => {
      await Promise.resolve();
    });

    const ipv4El = page.getByTestId("ipv4");
    await expect.element(ipv4El).toBeInTheDocument();
    await expect.element(ipv4El).toBeVisible();
    await expect.element(ipv4El).toHaveTextContent("empty");
    expect(queryIpv6Mock).not.toHaveBeenCalled();
  });

  it("keeps only the latest query results when a newer query supersedes an older one", async () => {
    const firstIpv4 = createDeferred<string[]>();
    const secondIpv4 = createDeferred<string[]>();
    const secondIpv6 = createDeferred<string[]>();

    queryIpv4Mock
      .mockImplementationOnce(async () => await firstIpv4.promise)
      .mockImplementationOnce(async () => await secondIpv4.promise);
    queryIpv6Mock.mockImplementationOnce(async () => await secondIpv6.promise);

    await mountHarness();

    await page.getByRole("button", { name: "Query First" }).click();
    await page.getByRole("button", { name: "Query Second" }).click();

    secondIpv4.resolve(["198.51.100.2"]);
    secondIpv6.resolve(["2001:db8::2"]);

    const ipv4El = page.getByTestId("ipv4");
    const ipv6El = page.getByTestId("ipv6");
    await expect.element(ipv4El).toHaveTextContent("198.51.100.2");
    await expect.element(ipv6El).toHaveTextContent("2001:db8::2");

    // Resolve the superseded first request after the second has completed
    firstIpv4.resolve(["192.0.2.1"]);
    await act(async () => {
      await Promise.resolve();
    });

    await expect.element(ipv4El).toBeInTheDocument();
    await expect.element(ipv4El).toBeVisible();
    await expect.element(ipv4El).toHaveTextContent("198.51.100.2");
    await expect.element(ipv6El).toHaveTextContent("2001:db8::2");
    await expect.element(page.getByTestId("loading")).toHaveTextContent("idle");
  });
});
