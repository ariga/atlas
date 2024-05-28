import React from "react";
import { Icon } from "./icon";

interface IDocLink {
  url: string;
  children: React.ReactNode;
  className?: string;
}

export function DocLink({ url, children, className }: IDocLink) {
  return (
    <a href={url} className={`text-base inline-flex items-center gap-2 ${className || ''}`}>
      {children} <Icon icon="arrow-right-blue.svg" />
    </a>
  );
}
