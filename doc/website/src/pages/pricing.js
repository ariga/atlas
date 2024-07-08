import React from "react";
import LayoutProvider from "@theme/Layout/Provider";
import AnnouncementBar from "@theme/AnnouncementBar";
import { AtlasGoPricing } from "@ariga/atlas-website";
import FOOTER from "../constants/footer";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <LayoutProvider>
      <AnnouncementBar />
      <AtlasGoPricing footer={FOOTER} />
    </LayoutProvider>
  );
}
