import React from "react";
import Layout from "@theme/Layout";
import { MigrationTool } from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <Layout title="Standardize Schema Migrations with Atlas">
      <MigrationTool />
    </Layout>
  );
}
