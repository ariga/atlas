import React from "react";

import { Icon } from "./icon";
import { Card } from "./card";

interface IDocCardProps {
  icon?: string;
  name: JSX.Element;
  url?: string;
  description: JSX.Element;
  className?: string;
}

export function DocCard({ name, icon, url, description, className }: IDocCardProps) {
  return (
    <Card url={url} className={`p-0 ${className || ""}`}>
      {icon ? <Icon icon={icon} className="w-full object-cover" /> : null}
      <div className="p-4 flex flex-col w-full">
        <h5 className="text-base font-bold text-black !mb-2 flex items-center">{name}</h5>
        <p className="text-black mb-0">{description}</p>
      </div>
    </Card>
  );
}
