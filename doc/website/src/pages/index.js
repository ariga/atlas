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
                  title: "Kubernetes-native schema migrations",  
                  date: "Wednesday, July 19th, 2023. 13:00 UTC",
                  description: "Discover the power of the Atlas Kubernetes Operator for seamless management of your database schemas in Kubernetes.",
                  linkUrl: 'http://26565461.hs-sites-eu1.com/kubernetes-native-schema-migrations-webinar',
                  imageUrl: 'https://atlasgo.io/uploads/images/posts/v0.3.2/kubernetes_webinar3.png',
                  linkText: 'Register',
              },
            ],
            isHidden: false,
        }}
        projectsAmount={2000}
      />
    </LayoutProvider>
  );
}

