export interface AppConfig {
  title: string;
  appId: string;
}

export const APP_CONFIG = {
  title: "Electron DNS",
  appId: "com.florencea.electrondns",
} as const satisfies AppConfig;
