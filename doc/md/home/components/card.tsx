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
        "flex flex-col items-center p-4 hover:no-underline dark:bg-[#fff] rounded-lg border-lightGrey border hover:shadow-md dark:hover:bg-lightGrey transition-all",
        className
      )}
    >
      {children}
    </Link>
  ) : (
    <div
      className={twMerge(
        "flex flex-col items-center p-4 rounded-lg dark:bg-[#fff] border-lightGrey border",
        className
      )}
    >
      {children}
    </div>
  );
}
