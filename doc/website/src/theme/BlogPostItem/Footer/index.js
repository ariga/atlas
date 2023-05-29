import React from "react";
import Footer from "@theme-original/BlogPostItem/Footer";
import { AtlasGoNewsletterDocs } from "@ariga/atlas-website";

export default function FooterWrapper(props) {
  return (
    <>
      <AtlasGoNewsletterDocs />
      <Footer {...props} />
    </>
  );
}
