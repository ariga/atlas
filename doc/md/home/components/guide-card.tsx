import React from "react";
import { Icon } from "./icon";
import { Card } from "./card";

interface IGuideCard {
  name: string;
  url: string;
  description: string;
  icon?: string;
  className?: string;
}

export function GuideCard({ name, url, description , icon, className }: IGuideCard) {
  return (
    <Card className={`items-start ${className || ""}`} url={url}>
      <h5 className="text-base font-bold text-black !mb-2 flex items-center">
        <Icon icon={icon ?? "guide-icon.svg"} className="mr-2" />
        {name}
      </h5>
      <p className="mb-0 text-black">{description}</p>
    </Card>
  );
}
