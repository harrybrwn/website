import underConstruction from "./under_construction.gif";

export const UnderConstruction = (
  width?: number | undefined,
  height?: number | undefined
): HTMLImageElement => {
  let im = new Image(width, height);
  im.src = underConstruction;
  return im;
};
