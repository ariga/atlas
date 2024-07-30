import React from "react";
import Layout from "@theme/Layout";
import { ModernizeSchemaMigrations } from "@ariga/atlas-website";

import "@ariga/atlas-website/style.css";

export default function () {
  return (
    <Layout>
      <ModernizeSchemaMigrations />
    </Layout>
  );
}
