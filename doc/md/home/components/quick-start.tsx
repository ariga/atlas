import React from "react";
import { Icon } from "./icon";
import Link from "@docusaurus/Link";

export function QuickStart({title, description, url}) {
  return (
      <Link
          to={url}
          className="hover:no-underline p-4 flex mb-12 flex-col gap-2 max-w-[300px] border border-[#A7B5FF] rounded-lg bg-[#E9ECFF] transition-all hover:border-lightBlue"
      >
        <div className="flex justify-between items-center">
          <p className="text-xl font-bold mb-0 text-black">{title}</p>
          <Icon icon="arrow-right.svg" />
        </div>
        <p className="mb-0 inline-block text-black">{description}</p>
      </Link>
  );
}
