import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ModeT, ServerT, SupportedLocale } from "../../src/shared/types";
import {
  initLocaleSubscription,
  resetLocaleStore,
} from "../../src/renderer/hooks/useLocale";
import { renderApp } from "./test-utils";

type QueryFn = (mode: ModeT, name: string) => Promise<string[]>;

const mockServers: ServerT[] = [
  {
    title: "Google Public DNS",
    description: "Google DNS resolver",
    document: "https://developers.google.com/speed/public-dns",
    server: "https://dns.google/resolve",
    ip: "8.8.8.8",
  },
  {
    title: "Pure DNS Provider",
    description: "Legacy DNS without DoH",
    document: "-",
    server: "-",
    ip: "1.2.3.4",
  },
];

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

describe("<App /> in Browser Mode", () => {
  let queryIpv4Mock: Mock<QueryFn>;
  let queryIpv6Mock: Mock<QueryFn>;
  let getLocaleMock: Mock<() => Promise<SupportedLocale>>;
  let setLocaleMock: Mock<(locale: SupportedLocale) => Promise<void>>;
  let localeChangeListeners: ((locale: SupportedLocale) => void)[];
  let onLocaleChangeMock: Mock<
    (callback: (locale: SupportedLocale) => void) => () => void
  >;

  beforeEach(() => {
    queryIpv4Mock = vi
      .fn<QueryFn>()
      .mockResolvedValue(["ipv4: 12.34ms\n93.184.216.34"]);
    queryIpv6Mock = vi
      .fn<QueryFn>()
      .mockResolvedValue(["ipv6: 15.67ms\n2606:2800:220:1:248:1893:25c8:1946"]);

    localeChangeListeners = [];
    getLocaleMock = vi
      .fn<() => Promise<SupportedLocale>>()
      .mockResolvedValue("en-US");
    setLocaleMock = vi
      .fn<(locale: SupportedLocale) => Promise<void>>()
      .mockImplementation(async (loc) => {
        await Promise.resolve();
        for (const listener of localeChangeListeners) {
          listener(loc);
        }
      });
    onLocaleChangeMock = vi
      .fn<(callback: (locale: SupportedLocale) => void) => () => void>()
      .mockImplementation((cb: (locale: SupportedLocale) => void) => {
        localeChangeListeners.push(cb);
        return () => {
          localeChangeListeners = localeChangeListeners.filter((l) => l !== cb);
        };
      });

    window.api = {
      servers: mockServers,
      queryIpv4: queryIpv4Mock,
      queryIpv6: queryIpv6Mock,
      getLocale: getLocaleMock,
      setLocale: setLocaleMock,
      onLocaleChange: onLocaleChangeMock,
    };

    resetLocaleStore("en-US");
    initLocaleSubscription();
  });

  it("renders server cards in default DNS mode", async () => {
    const screen = await renderApp();

    const googleHeading = screen.getByRole("heading", {
      name: "Google Public DNS",
      level: 1,
    });
    const pureDnsHeading = screen.getByRole("heading", {
      name: "Pure DNS Provider",
      level: 1,
    });

    await expect.element(googleHeading).toBeInTheDocument();
    await expect.element(googleHeading).toBeVisible();
    await expect.element(pureDnsHeading).toBeInTheDocument();
    await expect.element(pureDnsHeading).toBeVisible();
  });

  it("filters out non-DoH servers when switching mode to 'DNS over HTTPS'", async () => {
    const screen = await renderApp();

    const pureDnsHeading = screen.getByRole("heading", {
      name: "Pure DNS Provider",
      level: 1,
    });
    await expect.element(pureDnsHeading).toBeInTheDocument();

    // Click on the 'DNS over HTTPS' radio option
    const dohRadio = screen.getByRole("radio", { name: /DNS over HTTPS/i });
    await dohRadio.click();

    // Google Public DNS should remain visible
    const googleHeading = screen.getByRole("heading", {
      name: "Google Public DNS",
      level: 1,
    });
    await expect.element(googleHeading).toBeInTheDocument();

    // Pure DNS Provider should be filtered out
    await expect.element(pureDnsHeading).not.toBeInTheDocument();
  });

  it("triggers queries upon domain form submission and renders results", async () => {
    const screen = await renderApp();

    const submitBtn = screen.getByRole("button", { name: /submit|查詢/i });
    await expect.element(submitBtn).toBeVisible();

    const domainInput = screen.getByRole("textbox", {
      name: /domain|網域名稱/i,
    });
    await domainInput.fill("example.com");

    await submitBtn.click();

    expect(queryIpv4Mock).toHaveBeenCalledWith("dns", "example.com");
    expect(queryIpv6Mock).toHaveBeenCalledWith("dns", "example.com");

    const ipv4Output = screen.getByText(/93\.184\.216\.34/);
    await expect.element(ipv4Output).toBeInTheDocument();
    await expect.element(ipv4Output).toBeVisible();

    const ipv6Output = screen.getByText(/2606:2800:220:1:248:1893:25c8:1946/);
    await expect.element(ipv6Output).toBeInTheDocument();
    await expect.element(ipv6Output).toBeVisible();
  });

  it("clears rendered results locally without extra IPC when clicking Reset", async () => {
    const screen = await renderApp();

    const domainInput = screen.getByRole("textbox", {
      name: /domain|網域名稱/i,
    });
    await domainInput.fill("example.com");

    const submitBtn = screen.getByRole("button", { name: /submit|查詢/i });
    await submitBtn.click();

    const ipv4Output = screen.getByText(/93\.184\.216\.34/);
    await expect.element(ipv4Output).toBeInTheDocument();
    await expect.element(ipv4Output).toBeVisible();

    const resetBtn = screen.getByRole("button", { name: /reset|清除/i });
    await expect.element(resetBtn).toBeVisible();
    await resetBtn.click();

    await expect.element(ipv4Output).not.toBeInTheDocument();
    expect(queryIpv4Mock).toHaveBeenCalledTimes(1);
    expect(queryIpv6Mock).toHaveBeenCalledTimes(1);
  });

  it("disables the form while loading and renders IPv4 then IPv6 results", async () => {
    const deferredIpv4 = createDeferred<string[]>();
    const deferredIpv6 = createDeferred<string[]>();
    queryIpv4Mock.mockImplementationOnce(
      async () => await deferredIpv4.promise,
    );
    queryIpv6Mock.mockImplementationOnce(
      async () => await deferredIpv6.promise,
    );

    const screen = await renderApp();

    const domainInput = screen.getByRole("textbox", {
      name: /domain|網域名稱/i,
    });
    await domainInput.fill("slow.example.com");

    const submitBtn = screen.getByRole("button", { name: /submit|查詢/i });
    await submitBtn.click();

    await expect.element(domainInput).toBeDisabled();

    deferredIpv4.resolve(["ipv4: 10.00ms\n192.0.2.1"]);

    const ipv4Output = screen.getByText(/192\.0\.2\.1/);
    await expect.element(ipv4Output).toBeInTheDocument();
    await expect.element(ipv4Output).toBeVisible();

    deferredIpv6.resolve(["ipv6: 14.00ms\n2001:db8::1"]);

    const ipv6Output = screen.getByText(/2001:db8::1/);
    await expect.element(ipv6Output).toBeInTheDocument();
    await expect.element(ipv6Output).toBeVisible();
    await expect.element(domainInput).toBeEnabled();
  });

  it("recovers loading state when IPC query rejects", async () => {
    queryIpv4Mock.mockRejectedValueOnce(new Error("IPC failure"));

    const screen = await renderApp();

    const domainInput = screen.getByRole("textbox", {
      name: /domain|網域名稱/i,
    });
    await domainInput.fill("broken.example.com");

    const submitBtn = screen.getByRole("button", { name: /submit|查詢/i });
    await submitBtn.click();

    await expect.element(submitBtn).toBeEnabled();
    expect(queryIpv4Mock).toHaveBeenCalledWith("dns", "broken.example.com");
    expect(queryIpv6Mock).not.toHaveBeenCalled();
  });

  it("updates UI language dynamically when locale changes", async () => {
    const screen = await renderApp();

    const submitBtn = screen.getByRole("button", { name: /submit/i });
    await expect.element(submitBtn).toBeInTheDocument();

    for (const listener of localeChangeListeners) {
      listener("zh-TW");
    }

    const zhSubmitBtn = screen.getByRole("button", { name: /查詢/i });
    await expect.element(zhSubmitBtn).toBeInTheDocument();
    await expect.element(zhSubmitBtn).toBeVisible();
  });
});
