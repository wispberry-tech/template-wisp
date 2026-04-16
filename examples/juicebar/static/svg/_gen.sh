#!/bin/bash
# Generator script for juicebar placeholder SVGs.
# Re-run this if you want to tweak the palette; output files are committed.
set -euo pipefail

cd "$(dirname "$0")"

bottle() {
  local out=$1 fill=$2 accent=$3 label=$4
  cat > "$out" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 260" role="img" aria-label="$label">
  <rect width="200" height="260" fill="#fdf8f1"/>
  <g>
    <rect x="82" y="18" width="36" height="20" rx="3" fill="#3a2f1f"/>
    <rect x="70" y="38" width="60" height="22" rx="4" fill="#5c4a2f"/>
    <path d="M60 72 Q60 60 80 58 L120 58 Q140 60 140 72 L140 220 Q140 244 100 244 Q60 244 60 220 Z" fill="$fill"/>
    <path d="M70 84 Q70 78 82 76 L118 76 Q130 78 130 84 L130 216 Q130 232 100 232 Q70 232 70 216 Z" fill="#ffffff" opacity="0.18"/>
    <rect x="72" y="128" width="56" height="44" rx="4" fill="#fdf8f1" opacity="0.92"/>
    <text x="100" y="148" font-family="Georgia,serif" font-size="10" text-anchor="middle" fill="#1b1b1b">JUICEBAR</text>
    <text x="100" y="164" font-family="Georgia,serif" font-size="8" text-anchor="middle" fill="$accent">$label</text>
  </g>
</svg>
EOF
}

# products/
mkdir -p products
bottle products/fiery-ginger.svg            "#f77f2a" "#8f3d00" "GINGER"
bottle products/turmeric-sunrise.svg        "#f2c744" "#8a6a00" "TURMERIC"
bottle products/beetroot-charge.svg         "#a23658" "#5c1d31" "BEETROOT"
bottle products/classic-ginger-kombucha.svg "#c89a4d" "#5c3c0a" "KOMBUCHA"
bottle products/hibiscus-rose-kombucha.svg  "#c43a6e" "#6b1938" "HIBISCUS"
bottle products/blueberry-basil-kombucha.svg "#4a4db3" "#252566" "BLUEBERRY"
bottle products/green-machine.svg           "#6aa63a" "#365a17" "GREEN"
bottle products/citrus-sunrise.svg          "#f5a623" "#8a4d00" "CITRUS"
bottle products/pineapple-mint.svg          "#d9cf4f" "#6b6218" "PINEAPPLE"
bottle products/daily-three-pack.svg        "#f77f2a" "#8f3d00" "3-PACK"
bottle products/brew-six-pack.svg           "#c89a4d" "#5c3c0a" "6-PACK"
bottle products/reset-ritual.svg            "#6aa63a" "#365a17" "RITUAL"

# collections/
mkdir -p collections
bottle collections/boosters.svg     "#f77f2a" "#8f3d00" "BOOSTERS"
bottle collections/kombuchas.svg    "#c89a4d" "#5c3c0a" "KOMBUCHAS"
bottle collections/cold-pressed.svg "#6aa63a" "#365a17" "COLD-PRESS"
bottle collections/bundles.svg      "#c43a6e" "#6b1938" "BUNDLES"

# icons/ — simple flat stroke icons
mkdir -p icons
cat > icons/leaf.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 20c0-8 6-14 16-16-1 10-7 16-16 16Z"/><path d="M5 19 15 9"/></svg>
EOF
cat > icons/cart.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="9" cy="21" r="1.2"/><circle cx="18" cy="21" r="1.2"/><path d="M3 4h3l2 12h12l2-8H7"/></svg>
EOF
cat > icons/arrow.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M5 12h14M13 6l6 6-6 6"/></svg>
EOF
cat > icons/star.svg <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2 15 9l7 1-5 5 1 7-6-3-6 3 1-7-5-5 7-1Z"/></svg>
EOF

echo "Generated $(ls products collections icons | wc -l) files."
