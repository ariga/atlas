import React from "react";

import useBaseUrl from "@docusaurus/useBaseUrl";

interface IIconProps {
  icon: string;
  className?: string;
}

export function Icon({ icon, className }: IIconProps) {
  if (!icon) {
    return null;
  }
  return (
    <img
      className={`${className || ""}`}
      src={useBaseUrl(`/icons-docs/${icon}`)}
      alt={`${icon}'s image`}
    />
  );
}
