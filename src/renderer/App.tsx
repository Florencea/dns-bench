import { Flex, Provider, View, defaultTheme } from "@adobe/react-spectrum";
import { useState } from "react";
import { getTranslations } from "../shared/locales";
import type { ModeT } from "../shared/types";
import { HeaderBar } from "./components/HeaderBar";
import { QueryForm } from "./components/QueryForm";
import { ServerGrid } from "./components/ServerGrid";
import { useDnsQuery } from "./hooks/useDnsQuery";
import { useLocale } from "./hooks/useLocale";

export const App = () => {
  const [mode, setMode] = useState<ModeT>("dns");
  const [locale, handleLocaleChange] = useLocale();
  const { ipv4Data, ipv6Data, isLoading, runQuery, reset } = useDnsQuery();

  const t = getTranslations(locale);

  const handleSubmit = async (host: string) => {
    await runQuery(mode, host);
  };

  const handleReset = () => {
    reset();
  };

  const handleModeChange = (newMode: ModeT) => {
    setMode(newMode);
    reset();
  };

  return (
    <Provider
      locale={locale}
      theme={defaultTheme}
      UNSAFE_className="h-svh overflow-y-scroll"
    >
      <View padding="size-200" paddingTop="size-200">
        <Flex direction="column" gap="size-100">
          <HeaderBar
            locale={locale}
            onLocaleChange={(nextLocale) => {
              void handleLocaleChange(nextLocale);
            }}
            t={t.ui}
          />
          <QueryForm
            mode={mode}
            onModeChange={handleModeChange}
            onSubmit={handleSubmit}
            onReset={handleReset}
            isLoading={isLoading}
            t={t.ui}
          />
          <ServerGrid
            servers={window.api?.servers ?? []}
            mode={mode}
            ipv4Data={ipv4Data}
            ipv6Data={ipv6Data}
            t={t.ui}
          />
        </Flex>
      </View>
    </Provider>
  );
};
