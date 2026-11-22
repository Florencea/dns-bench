import { useCallback, useRef, useState } from "react";
import type { ModeT } from "../../shared/types";

export interface DnsQueryState {
  ipv4Data: string[] | undefined;
  ipv6Data: string[] | undefined;
  isLoading: boolean;
  runQuery: (mode: ModeT, host: string) => Promise<void>;
  reset: () => void;
}

export function useDnsQuery(): DnsQueryState {
  const [ipv4Data, setIpv4Data] = useState<string[] | undefined>(undefined);
  const [ipv6Data, setIpv6Data] = useState<string[] | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(false);
  const requestIdRef = useRef(0);

  const reset = useCallback(() => {
    requestIdRef.current += 1;
    setIpv4Data(undefined);
    setIpv6Data(undefined);
    setIsLoading(false);
  }, []);

  const runQuery = useCallback(async (mode: ModeT, host: string) => {
    const currentId = requestIdRef.current + 1;
    requestIdRef.current = currentId;

    setIpv4Data(undefined);
    setIpv6Data(undefined);
    setIsLoading(true);

    try {
      const nextIpv4 =
        window.api !== undefined ? await window.api.queryIpv4(mode, host) : [];
      if (requestIdRef.current !== currentId) {
        return;
      }
      setIpv4Data(nextIpv4);

      const nextIpv6 =
        window.api !== undefined ? await window.api.queryIpv6(mode, host) : [];
      if (requestIdRef.current !== currentId) {
        return;
      }
      setIpv6Data(nextIpv6);
    } catch {
      if (requestIdRef.current === currentId) {
        setIpv4Data([]);
        setIpv6Data([]);
      }
    } finally {
      if (requestIdRef.current === currentId) {
        setIsLoading(false);
      }
    }
  }, []);

  return { ipv4Data, ipv6Data, isLoading, runQuery, reset };
}
