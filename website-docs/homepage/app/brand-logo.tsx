import Image from "next/image";
import { homeAssets } from "../../shared/header";

/** Show the Rethra wordmark. */
export function BrandLogo({ priority = false }: { priority?: boolean }) {
  return <span className="wk-logo"><Image src={`${homeAssets}/brand/rethra-original.png`} alt="Rethra" width={800} height={248} priority={priority} /></span>;
}
