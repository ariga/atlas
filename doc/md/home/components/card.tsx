import React from "react";
import { twMerge } from "tailwind-merge";

import Link from "@docusaurus/Link";

interface ICard {
  url?: string;
  className?: string;
}

export function Card({ children, className, url }: React.ReactWithChildren<ICard>) {
  return url ? (
    <Link
      to={url}
      className={twMerge(
        "flex flex-col items-center p-4 hover:no-underline rounded-lg border-lightGrey border hover:shadow-md transition-all",
        className
      )}
    >
      {children}
    </Link>
  ) : (
    <div
      className={twMerge(
        "flex flex-col items-center p-4 rounded-lg border-lightGrey border",
        className
      )}
    >
      {children}
    </div>
  );
}
