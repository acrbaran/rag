"""Copy the transparent Rethra logo for both intro-video themes."""
from pathlib import Path
from shutil import copyfile

here = Path(__file__).resolve().parent
source = here.parents[1] / "docs/images/logo.png"
for theme in ("light", "dark"):
    copyfile(source, here / f"assets/logo-{theme}.png")
