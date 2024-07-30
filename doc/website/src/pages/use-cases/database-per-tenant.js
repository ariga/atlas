import React from "react";
import Layout from "@theme/Layout";
import { DatabasePerTenant } from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <Layout>
      <DatabasePerTenant />
    </Layout>
  );
}
