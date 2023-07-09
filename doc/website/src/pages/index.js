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
                  title: "HashiTalks 2023",
                  date: "Thursday, July 13th, 2023. 08:05 UTC",
                  description: "Join our lecture on HashiTalks:Israel 2023 and learn how to build modern schema change " +
                                "management workflows using Terraform, Vault, and Atlas to sustain fast and safe schema evolution.",
                  linkUrl: 'https://events.hashicorp.com/hashitalksisrael',
                  imageUrl: 'https://atlasgo.io/uploads/images/posts/v0.3.2/hashitialks2023_3.png',
                  linkText: 'Register',
              },
              {
                  title: "Kubernetes-native schema migrations",  
                  date: "Wednesday, July 19th, 2023. 13:00 UTC",
                  description: "Discover the power of the Atlas Kubernetes Operator for seamless management of your database schemas in Kubernetes.",
                  linkUrl: 'https://26565461.hs-sites-eu1.com/webinar-ci/cd-for-databases-with-atlas-cloud-0-0',
                  imageUrl: 'https://atlasgo.io/uploads/images/posts/v0.3.2/kubernetes_webinar3.png',
                  linkText: 'Register',
              },
            ],
            isHidden: false,
        }}
      />
    </LayoutProvider>
  );
}

