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
                  title: "CI/CD for Databases",
                  date: "Monday, June 12th, 2023. 13:00 UTC",
                  description: "Learn how to use Atlas Cloud to prevent risky " +
                      "database changes and apply CD for schema changes with Terraform and Kubernetes.",
                  linkUrl: "http://26565461.hs-sites-eu1.com/webinar-ci/cd-for-databases-with-atlas-cloud",
                  imageUrl: "https://atlasgo.io/uploads/event-placeholder.png",
              },
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
