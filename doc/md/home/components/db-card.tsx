import React from "react";
import { Icon } from "./icon";
import { Card } from "./card";

interface IDBCard {
  name: string;
  icon: string;
  url: string;
  isDark: boolean;
  children?: React.ReactNode;
}

export function DBCard({ name, url, icon, isDark, children }: IDBCard) {
  return (
    <Card
      url={url}
      className={`relative justify-center border-lightGrey border px-2 
      ${isDark ? "bg-lightBlue col-end-3 top-10 xl:top-0 xl:col-end-auto" : ""}`}
    >
      <div className="flex flex-col gap-2 items-center relative">
        <div className="absolute -top-14">
          <div className="w-20 flex justify-end">{children}</div>
        </div>
        <Icon icon={icon} className="w-8 h-8" />
        <p className={`text-center mb-0 ${isDark ? "!text-lightGrey" : "text-darkBlue"}`}>{name}</p>
      </div>
    </Card>
  );
}
