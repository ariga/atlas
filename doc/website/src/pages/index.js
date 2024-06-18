import React from "react";
import LayoutProvider from "@theme/Layout/Provider";
import AnnouncementBar from "@theme/AnnouncementBar";
import { AtlasGoWebsite } from "@ariga/atlas-website";
import FOOTER from "../constants/footer";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <LayoutProvider>
      <AnnouncementBar />
      <AtlasGoWebsite
        events={{
          events: [],
          isHidden: true,
        }}
        projectsAmount={3000}
        footer={FOOTER}
      />
    </LayoutProvider>
  );
}
