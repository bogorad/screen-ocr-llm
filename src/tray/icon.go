package tray

import (
	_ "embed"
)

// Embedded SVG icon data
//
//go:embed icon.svg
var IconSVG string

// SVG content for the tray icon
const SVGContent = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="16" height="16">
  <!-- Selection loop/rectangle -->
  <rect x="3" y="3" width="8" height="6" fill="none" stroke="#0078d4" stroke-width="1.5" stroke-dasharray="2,1" opacity="0.8"/>
  
  <!-- Scissors -->
  <g transform="translate(10.5, 11) rotate(-45)">
    <!-- Scissor handles -->
    <circle cx="0" cy="-1" r="1" fill="none" stroke="#333333" stroke-width="0.8"/>
    <circle cx="0" cy="1" r="1" fill="none" stroke="#333333" stroke-width="0.8"/>
    
    <!-- Scissor blades -->
    <line x1="0.7" y1="-0.3" x2="2.5" y2="-0.8" stroke="#333333" stroke-width="1" stroke-linecap="round"/>
    <line x1="0.7" y1="0.3" x2="2.5" y2="0.8" stroke="#333333" stroke-width="1" stroke-linecap="round"/>
    
    <!-- Center pivot -->
    <circle cx="0.5" cy="0" r="0.3" fill="#666666"/>
  </g>
  
  <!-- Small cut line to show action -->
  <line x1="8" y1="9.5" x2="10" y2="11.5" stroke="#666666" stroke-width="1" stroke-dasharray="1,1" opacity="0.6"/>
</svg>`
