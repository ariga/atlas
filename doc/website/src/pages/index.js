import React from "react";
import LayoutProvider from "@theme/Layout/Provider";
import * as Atlas from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <LayoutProvider>
      <Atlas.WebsiteWrapper>
        <Atlas.BackgroundWrapper>
          <Atlas.ContentWrapper>
            <Atlas.Header />
            <Atlas.Intro />
            <Atlas.CompaniesDesktop />
            <Atlas.UseCase />
            <Atlas.TestimonialsTablet />
            <Atlas.Guides />
            <Atlas.CompaniesMobile />
            <Atlas.Newsletter />
            <Atlas.Discover />
            <Atlas.Footer />
          </Atlas.ContentWrapper>
        </Atlas.BackgroundWrapper>
      </Atlas.WebsiteWrapper>
    </LayoutProvider>
  );
}
