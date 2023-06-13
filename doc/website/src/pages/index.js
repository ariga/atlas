import React from "react";
import LayoutProvider from "@theme/Layout/Provider";
import { AtlasGoWebsite } from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <LayoutProvider>
      <AtlasGoWebsite
        events={{
          events: [
              {
                  title: "Getting started with Atlas Cloud",
                  date: "COMING SOON",
                  description: "Join us to learn how to set up Atlas Cloud for your team in under one minute.",
                  imageUrl: "https://atlasgo.io/uploads/event-placeholder.png",
              },
            ],
            isHidden: false,
        }}
      />
    </LayoutProvider>
  );
}
