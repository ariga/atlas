import React from "react";
import LayoutProvider from "@theme/Layout/Provider";
import BrowserOnly from "@docusaurus/core/lib/client/exports/BrowserOnly";
import { AtlasWebsite } from "@ariga/atlas-website";
import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <LayoutProvider>
      <BrowserOnly>
        {() => {
          return <AtlasWebsite />;
        }}
      </BrowserOnly>
    </LayoutProvider>
  );
}
