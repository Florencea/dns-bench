import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { initLocaleSubscription } from "./hooks/useLocale";
import "./index.css";

initLocaleSubscription();

const container = document.getElementById("root");
if (container !== null) {
  createRoot(container).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
}
