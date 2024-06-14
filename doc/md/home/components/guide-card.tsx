import React from "react";
import { Icon } from "./icon";
import { Card } from "./card";

interface IGuideCard {
  name: string;
  url: string;
  description: string;
}

export function GuideCard({ name, url, description }: IGuideCard) {
  return (
    <Card className="items-start" url={url}>
      <h5 className="text-base font-bold text-black !mb-2 flex items-center">
        <Icon icon="guide-icon.svg" className="mr-2" />
        {name}
      </h5>
      <p className="mb-0 text-black">{description}</p>
    </Card>
  );
}
