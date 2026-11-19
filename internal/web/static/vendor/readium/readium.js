var ws=Object.defineProperty;var ot=(r,t)=>()=>(r&&(t=r(r=0)),t);var nt=(r,t)=>{for(var e in t)ws(r,e,{get:t[e],enumerable:!0})};var Wr={};nt(Wr,{default:()=>Wa});var Wa,Br=ot(()=>{Wa=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

@-ms-viewport{
  width:device-width;
}

@viewport{
  width:device-width;
  zoom:1;
}

:root{

  --RS__sans-serif-ja-v:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDGothic', 'Yu Gothic', '\uFF2D\uFF33\u30B4\u30B7\u30C3\u30AF', 'MS Gothic', sans-serif;

  --RS__serif-ja-v:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDMincho', 'Yu Mincho', '\uFF2D\uFF33\u660E\u671D', 'MS Mincho', serif;

  --RS__sans-serif-ja:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDPGothic', 'Yu Gothic', '\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF', 'MS PGothic', sans-serif;

  --RS__serif-ja:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDPMincho', 'Yu Mincho', '\uFF2D\uFF33 \uFF30\u660E\u671D', 'MS PMincho', serif;

  --RS__monospaceTf:ui-monospace, 'Andale Mono', 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;

  --RS__humanistTf:Seravek, Calibri, 'Gill Sans Nova', Roboto, Ubuntu, 'DejaVu Sans', source-sans-pro, sans-serif;

  --RS__sansTf:-ui-sans-serif, -apple-system, system-ui, BlinkMacSystemFont, 'Segoe UI Variable', 'Segoe UI', Inter, Roboto, 'Helvetica Neue', 'Arial Nova', 'Liberation Sans', Arial, sans-serif;

  --RS__modernTf:Athelas, Constantia, Charter, 'Bitstream Charter', Cambria, 'Georgia Pro', Georgia, serif;

  --RS__oldStyleTf:'Iowan Old Style', Sitka, 'Sitka Text', Palatino, 'Book Antiqua', 'URW Palladio L', P052, serif;

  --RS__zh-HK-lineHeightCompensation:1.167;

  --RS__zh-HK-baseFontFamily:'\u65B9\u9AD4', 'PingFang HK', '\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-TW-lineHeightCompensation:1.167;

  --RS__zh-TW-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-Hant-lineHeightCompensation:1.167;

  --RS__zh-Hant-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-lineHeightCompensation:1.167;

  --RS__zh-baseFontFamily:'\u65B9\u4F53', 'PingFang SC', '\u9ED1\u4F53', 'Heiti SC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK SC', sans-serif;

  --RS__th-lineHeightCompensation:1.067;

  --RS__th-baseFontFamily:Thonburi, 'Leelawadee UI', 'Cordia New', Roboto, Noto, 'Noto Sans Thai', sans-serif;

  --RS__te-baseFontFamily:'Kohinoor Telugu', 'Telugu Sangam MN', 'Nirmala UI', Gautami, Roboto, Noto, 'Noto Sans Telugu', sans-serif;

  --RS__ta-lineHeightCompensation:1.067;

  --RS__ta-baseFontFamily:'Tamil Sangam MN', 'Nirmala UI', Latha, Roboto, Noto, 'Noto Sans Tamil', sans-serif;

  --RS__si-lineHeightCompensation:1.167;

  --RS__si-baseFontFamily:'Sinhala Sangam MN', 'Nirmala UI', 'Iskoola Pota', Roboto, Noto, 'Noto Sans Sinhala', sans-serif;

  --RS__pa-lineHeightCompensation:1.1;

  --RS__pa-baseFontFamily:'Gurmukhi MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Gurmukhi', sans-serif;

  --RS__or-lineHeightCompensation:1.167;

  --RS__or-baseFontFamily:'Oriya Sangam MN', 'Nirmala UI', Kalinga, Roboto, Noto, 'Noto Sans Oriya', sans-serif;

  --RS__ml-lineHeightCompensation:1.067;

  --RS__ml-baseFontFamily:'Malayalam Sangam MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Malayalam', sans-serif;

  --RS__lo-baseFontFamily:'Lao Sangam MN', 'Leelawadee UI', 'Lao UI', Roboto, Noto, 'Noto Sans Lao', sans-serif;

  --RS__ko-lineHeightCompensation:1.167;

  --RS__ko-baseFontFamily:'Nanum Gothic', 'Apple SD Gothic Neo', 'Malgun Gothic', Roboto, Noto, 'Noto Sans CJK KR', sans-serif;

  --RS__kn-lineHeightCompensation:1.1;

  --RS__kn-baseFontFamily:'Kannada Sangam MN', 'Nirmala UI', Tunga, Roboto, Noto, 'Noto Sans Kannada', sans-serif;

  --RS__km-lineHeightCompensation:1.067;

  --RS__km-baseFontFamily:'Khmer Sangam MN', 'Leelawadee UI', 'Khmer UI', Roboto, Noto, 'Noto Sans Khmer', sans-serif;

  --RS__ja-lineHeightCompensation:1.167;

  --RS__ja-baseFontFamily:YuGothic, 'Hiragino Maru Gothic ProN', 'Hiragino Sans', 'Yu Gothic UI', 'Meiryo UI', 'MS Gothic', Roboto, Noto, 'Noto Sans CJK JP', sans-serif;

  --RS__iu-baseFontFamily:'Euphemia UCAS', Euphemia, Roboto, Noto, 'Noto Sans Canadian Aboriginal', sans-serif;

  --RS__hy-baseFontFamily:Mshtakan, Sylfaen, Roboto, Noto, 'Noto Serif Armenian', serif;

  --RS__hi-lineHeightCompensation:1.1;

  --RS__hi-baseFontFamily:'Kohinoor Devanagari', 'Devanagari Sangam MN', Kokila, 'Nirmala UI', Roboto, Noto, 'Noto Sans Devanagari', sans-serif;

  --RS__he-lineHeightCompensation:1.1;

  --RS__he-baseFontFamily:'New Peninim MT', 'Arial Hebrew', Gisha, 'Times New Roman', Roboto, Noto, 'Noto Sans Hebrew', sans-serif;

  --RS__gu-lineHeightCompensation:1.167;

  --RS__gu-baseFontFamily:'Gujarati Sangam MN', 'Nirmala UI', Shruti, Roboto, Noto, 'Noto Sans Gujarati', sans-serif;

  --RS__fa-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__chr-lineHeightCompensation:1.167;

  --RS__chr-baseFontFamily:'Plantagenet Cherokee', Roboto, Noto, 'Noto Sans Cherokee';

  --RS__bo-baseFontFamily:Kailasa, 'Microsoft Himalaya', Roboto, Noto, 'Noto Sans Tibetan', sans-serif;

  --RS__bn-lineHeightCompensation:1.067;

  --RS__bn-baseFontFamily:'Kohinoor Bangla', 'Bangla Sangam MN', Vrinda, Roboto, Noto, 'Noto Sans Bengali', sans-serif;

  --RS__ar-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__am-lineHeightCompensation:1.167;

  --RS__am-baseFontFamily:Kefa, Nyala, Roboto, Noto, 'Noto Sans Ethiopic', serif;

  --RS__latin-lineHeightCompensation:1;

  --RS__latin-baseFontFamily:var(--RS__oldStyleTf);
  --RS__baseFontFamily:var(--RS__latin-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__latin-lineHeightCompensation);
  --RS__baseLineHeight:calc(1.5 * var(--RS__lineHeightCompensation));

  --RS__selectionTextColor:inherit;

  --RS__selectionBackgroundColor:#b4d8fe;

  --RS__visitedColor:#551A8B;

  --RS__linkColor:#0000EE;

  --RS__textColor:#121212;

  --RS__backgroundColor:#FFFFFF;
  color:var(--RS__textColor) !important;

  background-color:var(--RS__backgroundColor) !important;
}

::-moz-selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

::selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

html{
  font-family:var(--RS__baseFontFamily);
  line-height:1.6;
  line-height:var(--RS__baseLineHeight);
  text-rendering:optimizelegibility;
}

h1, h2, h3{
  line-height:normal;
}

:lang(ja),
:lang(zh),
:lang(ko){
  word-wrap:break-word;
  -webkit-line-break:strict;
  -epub-line-break:strict;
  line-break:strict;
}

math{
  font-family:"Latin Modern Math", "STIX Two Math", "XITS Math", "STIX Math", "Libertinus Math", "TeX Gyre Termes Math", "TeX Gyre Bonum Math", "TeX Gyre Schola", "DejaVu Math TeX Gyre", "TeX Gyre Pagella Math", "Asana Math", "Cambria Math", "Lucida Bright Math", "Minion Math", STIXGeneral, STIXSizeOneSym, Symbol, "Times New Roman", serif;
}

:lang(am){
  --RS__baseFontFamily:var(--RS__am-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__am-lineHeightCompensation);
}

:lang(ar){
  --RS__baseFontFamily:var(--RS__ar-baseFontFamily);
}

:lang(bn){
  --RS__baseFontFamily:var(--RS__bn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__bn-lineHeightCompensation);
}

:lang(bo){
  --RS__baseFontFamily:var(--RS__bo-baseFontFamily);
}

:lang(chr){
  --RS__baseFontFamily:var(--RS__chr-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__chr-lineHeightCompensation);
}

:lang(fa){
  --RS__baseFontFamily:var(--RS__fa-baseFontFamily);
}

:lang(gu){
  --RS__baseFontFamily:var(--RS__gu-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__gu-lineHeightCompensation);
}

:lang(he){
  --RS__baseFontFamily:var(--RS__he-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__he-lineHeightCompensation);
}

:lang(hi){
  --RS__baseFontFamily:var(--RS__hi-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__hi-lineHeightCompensation);
}

:lang(hy){
  --RS__baseFontFamily:var(--RS__hy-baseFontFamily);
}

:lang(iu){
  --RS__baseFontFamily:var(--RS__iu-baseFontFamily);
}

:lang(ja){
  --RS__baseFontFamily:var(--RS__ja-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ja-lineHeightCompensation);
}

:lang(km){
  --RS__baseFontFamily:var(--RS__km-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__km-lineHeightCompensation);
}

:lang(kn){
  --RS__baseFontFamily:var(--RS__kn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__kn-lineHeightCompensation);
}

:lang(ko){
  --RS__baseFontFamily:var(--RS__ko-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ko-lineHeightCompensation);
}

:lang(lo){
  --RS__baseFontFamily:var(--RS__lo-baseFontFamily);
}

:lang(ml){
  --RS__baseFontFamily:var(--RS__ml-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ml-lineHeightCompensation);
}

:lang(or){
  --RS__baseFontFamily:var(--RS__or-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__or-lineHeightCompensation);
}

:lang(pa){
  --RS__baseFontFamily:var(--RS__pa-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__pa-lineHeightCompensation);
}

:lang(si){
  --RS__baseFontFamily:var(--RS__si-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__si-lineHeightCompensation);
}

:lang(ta){
  --RS__baseFontFamily:var(--RS__ta-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ta-lineHeightCompensation);
}

:lang(te){
  --RS__baseFontFamily:var(--RS__te-baseFontFamily);
}

:lang(th){
  --RS__baseFontFamily:var(--RS__th-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__th-lineHeightCompensation);
}

:lang(zh){
  --RS__baseFontFamily:var(--RS__zh-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-lineHeightCompensation);
}

:lang(zh-Hant){
  --RS__baseFontFamily:var(--RS__zh-Hant-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-Hant-lineHeightCompensation);
}

:lang(zh-TW){
  --RS__baseFontFamily:var(--RS__zh-TW-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-TW-lineHeightCompensation);
}

:lang(zh-HK){
  --RS__baseFontFamily:var(--RS__zh-HK-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-HK-lineHeightCompensation);
}

body{
  widows:2;
  orphans:2;
}

figcaption, th, td{
  widows:1;
  orphans:1;
}

h2, h3, h4, h5, h6, dt,
hr, caption{
  -webkit-column-break-after:avoid;
  page-break-after:avoid;
  break-after:avoid;
}

h1, h2, h3, h4, h5, h6, dt,
figure, tr{
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

body{
  -webkit-hyphenate-character:"\\002D";
  -moz-hyphenate-character:"\\002D";
  -ms-hyphenate-character:"\\002D";
  hyphenate-character:"\\002D";
  -webkit-hyphenate-limit-lines:3;
  -ms-hyphenate-limit-lines:3;
  hyphenate-limit-lines:3;
}

h1, h2, h3, h4, h5, h6, dt,
figcaption, pre, caption, address,
center, code, var{
  -ms-hyphens:none;
  -moz-hyphens:none;
  -webkit-hyphens:none;
  -epub-hyphens:none;
  hyphens:none;
}

body{
  font-variant-numeric:oldstyle-nums proportional-nums;
}

:lang(ja) body,
:lang(zh) body,
:lang(ko) body{
  font-variant-numeric:lining-nums proportional-nums;
}

h1, h2, h3, h4, h5, h6, dt{
  font-variant-numeric:lining-nums proportional-nums;
}

table{
  font-variant-numeric:lining-nums tabular-nums;
}

code, var{
  font-variant-ligatures:none;
  font-variant-numeric:lining-nums tabular-nums slashed-zero;
}

rt{
  font-variant-east-asian:ruby;
}

:lang(ar){
  font-variant-ligatures:common-ligatures;
}

:lang(ko){
  font-kerning:normal;
}

hr{
  color:inherit;
  border-color:currentcolor;
}

table, th, td{
  border-color:currentcolor;
}

figure, blockquote{
  margin:1em 5%;
}

ul, ol{
  padding-left:5%;
}

dd{
  margin-left:5%;
}

pre{
  white-space:pre-wrap;
  -ms-tab-size:2;
  -moz-tab-size:2;
  -webkit-tab-size:2;
  tab-size:2;
}

abbr[title], acronym[title]{
  text-decoration:dotted underline;
}

nobr wbr{
  white-space:normal;
}

ruby > rt, ruby > rp{
  -webkit-user-select:none;
  -moz-user-select:none;
  -ms-user-select:none;
  user-select:none;
}

*:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)),
*:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)),
*:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)),
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) cite, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) dfn, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) em, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) i,
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) cite, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) dfn, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) em, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) i,
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) cite, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) dfn, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) em, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) i{
  font-style:normal;
}

:lang(ja) a,
:lang(zh) a,
:lang(ko) a{
  text-decoration:none;
}

:root{
  --RS__maxMediaWidth:100%;
  --RS__maxMediaHeight:95vh;
  --RS__boxSizingMedia:border-box;
  --RS__boxSizingTable:border-box;
}

a, a span, span a, h1, h2, h3, h4, h5, h6{
  word-wrap:break-word;
}

div{
  max-width:var(--RS__maxMediaWidth);
}

img, svg|svg, video{
  object-fit:contain;

  width:auto;
  height:auto;
  max-width:var(--RS__maxMediaWidth);
  max-height:var(--RS__maxMediaHeight) !important;
  box-sizing:var(--RS__boxSizingMedia);
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

@supports (zoom: 1) and (not ((-webkit-column-axis: horizontal) and (-webkit-column-progression: normal))){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] img,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] svg|svg,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] video{
    zoom:calc(100% / var(--USER__fontSize));
  }
}

audio{
    max-width:100%;
    -webkit-column-break-inside:avoid;
    page-break-inside:avoid;
    break-inside:avoid;
  }

table{
  max-width:var(--RS__maxMediaWidth);
  box-sizing:var(--RS__boxSizingTable);
}`});var jr={};nt(jr,{default:()=>Ba});var Ba,Gr=ot(()=>{Ba=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{
  --RS__compFontFamily:var(--RS__baseFontFamily);
  --RS__codeFontFamily:var(--RS__monospaceTf);

  --RS__typeScale:1.125;
  --RS__baseFontSize:100%;

  --RS__flowSpacing:1.5rem;
  --RS__paraSpacing:0;
  --RS__paraIndent:1em;

  --RS__linkColor:#0000EE;
  --RS__visitedColor:#551A8B;

  --RS__primaryColor:;
  --RS__secondaryColor:;
}

body{
  font-size:var(--RS__baseFontSize);
  text-align:justify;
}

h1, h2, h3, h4, h5, h6{
  font-family:var(--RS__compFontFamily);
  text-align:right;
}

blockquote, figure, p, pre,
aside, footer, form, hr{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

p{
  margin-top:var(--RS__paraSpacing);
  margin-bottom:var(--RS__paraSpacing);
  text-indent:var(--RS__paraIndent);
}

h1 + p, h2 + p, h3 + p, h4 + p, h5 + p, h6 + p,
hr + p{
  text-indent:0;
}

pre{
  font-family:var(--RS__codeFontFamily);
}

code, kbd, samp, tt{
  font-family:var(--RS__codeFontFamily);
}

sub, sup{
  position:relative;
  font-size:67.5%;
  line-height:1;
}

sub{
  bottom:-0.2ex;
}

sup{
  bottom:0;
}

:link{
  color:var(--RS__linkColor);
}

:visited{
  color:var(--RS__visitedColor);
}

h1{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:calc(var(--RS__flowSpacing) * 2);
  font-size:calc(((1em * var(--RS__typeScale)) * var(--RS__typeScale)) * var(--RS__typeScale));
}

h2{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc((1em * var(--RS__typeScale)) * var(--RS__typeScale));
}

h3{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc(1em * var(--RS__typeScale));
}

h4{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:1em;
}

h5{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:smaller;
}

h6{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:0;
  font-size:smaller;
  font-weight:normal;
}

dl, ol, ul{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

table{
  margin:var(--RS__flowSpacing) 0;
  border:1px solid currentcolor;
  border-collapse:collapse;
  empty-cells:show;
}

thead, tbody, tfoot, table > tr{
  vertical-align:top;
}

th{
  text-align:initial;
}

th, td{
  padding:4px;
  border:1px solid currentcolor;
}`});var Vr={};nt(Vr,{default:()=>ja});var ja,Xr=ot(()=>{ja=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

  --RS__viewportWidth:100%;

  --RS__pageGutter:0;

  --RS__defaultLineLength:100%;

  --RS__colGap:0;

  --RS__colCount:1;

  --RS__colWidth:100vw;
}

@page{
  margin:0 !important;
}

:root{
  position:relative;

  -webkit-column-width:var(--RS__colWidth);
  -moz-column-width:var(--RS__colWidth);
  column-width:var(--RS__colWidth);
  -webkit-column-count:var(--RS__colCount);
  -moz-column-count:var(--RS__colCount);
  column-count:var(--RS__colCount);

  -webkit-column-gap:var(--RS__colGap);
  -moz-column-gap:var(--RS__colGap);
  column-gap:var(--RS__colGap);
  -moz-column-fill:auto;
  column-fill:auto;
  width:var(--RS__viewportWidth);
  height:100vh;
  max-width:var(--RS__viewportWidth);
  max-height:100vh;
  min-width:var(--RS__viewportWidth);
  min-height:100vh;
  padding:0 !important;
  margin:0 !important;
  font-size:1rem !important;
  box-sizing:border-box;
  -webkit-touch-callout:none;
}

body{
  width:100%;
  max-width:var(--RS__defaultLineLength) !important;
  margin:0 auto !important;
  box-sizing:border-box;
}

:root:not([style*="readium-scroll-on"]) body{
  padding:0 var(--RS__pageGutter) !important;
}

:root:not([style*="readium-noOverflow-on"]) body{
  overflow:hidden;
}

@supports (overflow: clip){

   :root:not([style*="readium-noOverflow-on"]){
      overflow:clip;
   }

   :root:not([style*="readium-noOverflow-on"]) body{
      overflow:clip;
      overflow-clip-margin:content-box;
   }
}

:root[style*="readium-scroll-on"]{
  -webkit-columns:auto auto !important;
  -moz-columns:auto auto !important;
  columns:auto auto !important;
  width:auto !important;
  height:auto !important;
  max-width:none !important;
  max-height:none !important;
  min-width:0 !important;
  min-height:0 !important;
}

:root[style*="readium-scroll-on"] body{
  max-width:var(--RS__defaultLineLength) !important;
  box-sizing:border-box !important;
}

:root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
  overflow:auto;
}

@supports (overflow: clip){

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]){
     overflow:auto;
  }

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
     overflow:clip;
  }
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingTop"] body{
  padding-top:var(--RS__scrollPaddingTop) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingBottom"] body{
  padding-bottom:var(--RS__scrollPaddingBottom) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingLeft"] body{
  padding-left:var(--RS__scrollPaddingLeft) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingRight"] body{
  padding-right:var(--RS__scrollPaddingRight) !important;
}

:root[style*="--USER__backgroundColor"]{
  background-color:var(--USER__backgroundColor) !important;
}

:root[style*="--USER__backgroundColor"] *{
  background-color:transparent !important;
}

:root[style*="--USER__textColor"]{
  color:var(--USER__textColor) !important;
}

:root[style*="--USER__textColor"] *:not(a){
  color:inherit !important;
  background-color:transparent !important;
  border-color:currentcolor !important;
}

:root[style*="--USER__textColor"] svg text{
  fill:currentcolor !important;
  stroke:none !important;
}

:root[style*="--USER__linkColor"] a:link,
:root[style*="--USER__linkColor"] a:link *{
  color:var(--USER__linkColor) !important;
}

:root[style*="--USER__visitedColor"] a:visited,
:root[style*="--USER__visitedColor"] a:visited *{
  color:var(--USER__visitedColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::-moz-selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__colCount"]{
  -webkit-column-count:var(--USER__colCount);
  -moz-column-count:var(--USER__colCount);
  column-count:var(--USER__colCount);

  --RS__colWidth:auto;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"]{
  -webkit-column-count:1;
  -moz-column-count:1;
  column-count:1;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"],
:root[style*="--USER__colCount: 1"],
:root[style*="--USER__colCount:1"]{
  --RS__colWidth:100vw;
}

:root[style*="--USER__lineLength"] body{
    max-width:var(--USER__lineLength) !important;
  }

:root[style*="--USER__textAlign"]{
  text-align:var(--USER__textAlign);
}

:root[style*="--USER__textAlign"] body,
:root[style*="--USER__textAlign"] p:not(
  blockquote p,
  figcaption p,
  header p,
  hgroup p,
  :root[style*="readium-experimentalHeaderFiltering-on"] p[class*="title"],
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > h1 + p,
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > p:has(+ h1)
),
:root[style*="--USER__textAlign"] li,
:root[style*="--USER__textAlign"] dd{
  text-align:var(--USER__textAlign) !important;
  -moz-text-align-last:auto !important;
  -epub-text-align-last:auto !important;
  text-align-last:auto !important;
}

:root[style*="--USER__fontFamily"]{
  font-family:var(--USER__fontFamily) !important;
}

:root[style*="--USER__fontFamily"] *{
  font-family:revert !important;
}

:root:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] body{
  zoom:var(--USER__fontSize) !important;
}

:root:not([style*="readium-deprecatedFontSize-on"])[style*="readium-iOSPatch-on"][style*="--USER__fontSize"] body{
  -webkit-text-size-adjust:var(--USER__fontSize) !important;
}

@supports selector(figure:has(> img)){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> img),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> video),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> svg),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> canvas),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> iframe),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> audio),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> img:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> video:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> svg:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> canvas:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> iframe:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> audio:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] table{
    zoom:calc(100% / var(--USER__fontSize)) !important;
  }

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figcaption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] caption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] td,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] th{
    zoom:var(--USER__fontSize) !important;
  }
}

@supports not (zoom: 1){

  :root[style*="--USER__fontSize"]{
    font-size:var(--USER__fontSize) !important;
  }
}

:root[style*="readium-deprecatedFontSize-on"][style*="--USER__fontSize"]{
  font-size:var(--USER__fontSize) !important;
}

:root[style*="--USER__lineHeight"]{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__lineHeight"] body,
:root[style*="--USER__lineHeight"] p,
:root[style*="--USER__lineHeight"] li,
:root[style*="--USER__lineHeight"] div{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__paraSpacing"] p{
  margin-top:var(--USER__paraSpacing) !important;
  margin-bottom:var(--USER__paraSpacing) !important;
}

:root[style*="--USER__paraIndent"] p:not(
  blockquote p,
  figcaption p,
  header p,
  hgroup p,
  :root[style*="readium-experimentalHeaderFiltering-on"] p[class*="title"],
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > h1 + p,
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > p:has(+ h1)
){
  text-indent:var(--USER__paraIndent) !important;
}

:root[style*="--USER__paraIndent"] p *{
  text-indent:0 !important;
}

:root[style*="--USER__wordSpacing"] h1,
:root[style*="--USER__wordSpacing"] h2,
:root[style*="--USER__wordSpacing"] h3,
:root[style*="--USER__wordSpacing"] h4,
:root[style*="--USER__wordSpacing"] h5,
:root[style*="--USER__wordSpacing"] h6,
:root[style*="--USER__wordSpacing"] p,
:root[style*="--USER__wordSpacing"] li,
:root[style*="--USER__wordSpacing"] div,
:root[style*="--USER__wordSpacing"] dt,
:root[style*="--USER__wordSpacing"] dd{
  word-spacing:var(--USER__wordSpacing) !important;
}

:root[style*="--USER__ligatures"]{
  font-variant-ligatures:var(--USER__ligatures) !important;
}

:root[style*="--USER__ligatures"] *{
  font-variant-ligatures:inherit !important;
}

:root[style*="--USER__fontWeight"] body{
  font-weight:var(--USER__fontWeight) !important;
}

:root[style*="--USER__fontWeight"] b,
:root[style*="--USER__fontWeight"] strong{
  font-weight:bolder;
}

:root[style*="--USER__fontWidth"] body{
  font-stretch:var(--USER__fontWidth) !important;
}

:root[style*="--USER__fontOpticalSizing"] body{
  font-optical-sizing:var(--USER__fontOpticalSizing) !important;
}

:root[style*="readium-blend-on"] svg,
:root[style*="readium-blend-on"] img{
  background-color:transparent !important;
  mix-blend-mode:multiply !important;
}

:root[style*="--USER__darkenImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) !important;
  filter:brightness(var(--USER__darkenImages)) !important;
}

:root[style*="readium-darken-on"] img{
  -webkit-filter:brightness(80%) !important;
  filter:brightness(80%) !important;
}

:root[style*="--USER__invertImages"] img{
  -webkit-filter:invert(var(--USER__invertImages)) !important;
  filter:invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-invert-on"] img{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="--USER__darkenImages"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
  filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-darken-on"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(80%) invert(var(--USER__invertImages)) !important;
  filter:brightness(80%) invert(var(--USER__invertImages)) !important;
}

:root[style*="--USER__darkenImages"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
  filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
}

:root[style*="readium-darken-on"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(80%) invert(100%) !important;
  filter:brightness(80%) invert(100%) !important;
}

:root[style*="--USER__invertGaiji"] img[class*="gaiji"]{
  -webkit-filter:invert(var(--USER__invertGaiji)) !important;
  filter:invert(var(--USER__invertGaiji)) !important;
}

:root[style*="readium-invertGaiji-on"] img[class*="gaiji"]{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="readium-normalize-on"]{
  --USER__typeScale:1.2;
}

:root[style*="readium-normalize-on"] p,
:root[style*="readium-normalize-on"] li,
:root[style*="readium-normalize-on"] div,
:root[style*="readium-normalize-on"] pre,
:root[style*="readium-normalize-on"] dd{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] h1{
  font-size:1.75rem !important;
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h2{
  font-size:1.5rem !important;
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h3{
  font-size:1.25rem !important;
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h4,
:root[style*="readium-normalize-on"] h5,
:root[style*="readium-normalize-on"] h6{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] small{
  font-size:smaller !important;
}

:root[style*="readium-normalize-on"] sub,
:root[style*="readium-normalize-on"] sup{
  font-size:67.5% !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h1{
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h2{
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h3{
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-iPadOSPatch-on"] body{
  -webkit-text-size-adjust:none;
}

:root[style*="readium-iPadOSPatch-on"] p, 
:root[style*="readium-iPadOSPatch-on"] h1, 
:root[style*="readium-iPadOSPatch-on"] h2, 
:root[style*="readium-iPadOSPatch-on"] h3, 
:root[style*="readium-iPadOSPatch-on"] h4, 
:root[style*="readium-iPadOSPatch-on"] h5, 
:root[style*="readium-iPadOSPatch-on"] h6, 
:root[style*="readium-iPadOSPatch-on"] li, 
:root[style*="readium-iPadOSPatch-on"] th, 
:root[style*="readium-iPadOSPatch-on"] td, 
:root[style*="readium-iPadOSPatch-on"] dt, 
:root[style*="readium-iPadOSPatch-on"] dd, 
:root[style*="readium-iPadOSPatch-on"] pre, 
:root[style*="readium-iPadOSPatch-on"] address, 
:root[style*="readium-iPadOSPatch-on"] details, 
:root[style*="readium-iPadOSPatch-on"] summary,
:root[style*="readium-iPadOSPatch-on"] figcaption,
:root[style*="readium-iPadOSPatch-on"] div:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)),
:root[style*="readium-iPadOSPatch-on"] aside:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)){
  -webkit-text-zoom:reset;
}

:root[style*="readium-iPadOSPatch-on"] abbr, 
:root[style*="readium-iPadOSPatch-on"] b, 
:root[style*="readium-iPadOSPatch-on"] bdi, 
:root[style*="readium-iPadOSPatch-on"] bdo, 
:root[style*="readium-iPadOSPatch-on"] cite, 
:root[style*="readium-iPadOSPatch-on"] code, 
:root[style*="readium-iPadOSPatch-on"] dfn, 
:root[style*="readium-iPadOSPatch-on"] em, 
:root[style*="readium-iPadOSPatch-on"] i, 
:root[style*="readium-iPadOSPatch-on"] kbd, 
:root[style*="readium-iPadOSPatch-on"] mark, 
:root[style*="readium-iPadOSPatch-on"] q, 
:root[style*="readium-iPadOSPatch-on"] rp, 
:root[style*="readium-iPadOSPatch-on"] rt, 
:root[style*="readium-iPadOSPatch-on"] ruby, 
:root[style*="readium-iPadOSPatch-on"] s, 
:root[style*="readium-iPadOSPatch-on"] samp, 
:root[style*="readium-iPadOSPatch-on"] small, 
:root[style*="readium-iPadOSPatch-on"] span, 
:root[style*="readium-iPadOSPatch-on"] strong, 
:root[style*="readium-iPadOSPatch-on"] sub, 
:root[style*="readium-iPadOSPatch-on"] sup, 
:root[style*="readium-iPadOSPatch-on"] time, 
:root[style*="readium-iPadOSPatch-on"] u, 
:root[style*="readium-iPadOSPatch-on"] var{
  -webkit-text-zoom:normal;
}

:root[style*="readium-iPadOSPatch-on"] p:not(:has(b, cite, em, i, q, s, small, span, strong)):first-line{
  -webkit-text-zoom:normal;
}`});var $r={};nt($r,{default:()=>Ga});var Ga,Kr=ot(()=>{Ga=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

@-ms-viewport{
  width:device-width;
}

@viewport{
  width:device-width;
  zoom:1;
}

:root{

  --RS__sans-serif-ja-v:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDGothic', 'Yu Gothic', '\uFF2D\uFF33\u30B4\u30B7\u30C3\u30AF', 'MS Gothic', sans-serif;

  --RS__serif-ja-v:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDMincho', 'Yu Mincho', '\uFF2D\uFF33\u660E\u671D', 'MS Mincho', serif;

  --RS__sans-serif-ja:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDPGothic', 'Yu Gothic', '\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF', 'MS PGothic', sans-serif;

  --RS__serif-ja:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDPMincho', 'Yu Mincho', '\uFF2D\uFF33 \uFF30\u660E\u671D', 'MS PMincho', serif;

  --RS__monospaceTf:ui-monospace, 'Andale Mono', 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;

  --RS__humanistTf:Seravek, Calibri, 'Gill Sans Nova', Roboto, Ubuntu, 'DejaVu Sans', source-sans-pro, sans-serif;

  --RS__sansTf:-ui-sans-serif, -apple-system, system-ui, BlinkMacSystemFont, 'Segoe UI Variable', 'Segoe UI', Inter, Roboto, 'Helvetica Neue', 'Arial Nova', 'Liberation Sans', Arial, sans-serif;

  --RS__modernTf:Athelas, Constantia, Charter, 'Bitstream Charter', Cambria, 'Georgia Pro', Georgia, serif;

  --RS__oldStyleTf:'Iowan Old Style', Sitka, 'Sitka Text', Palatino, 'Book Antiqua', 'URW Palladio L', P052, serif;

  --RS__zh-HK-lineHeightCompensation:1.167;

  --RS__zh-HK-baseFontFamily:'\u65B9\u9AD4', 'PingFang HK', '\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-TW-lineHeightCompensation:1.167;

  --RS__zh-TW-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-Hant-lineHeightCompensation:1.167;

  --RS__zh-Hant-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-lineHeightCompensation:1.167;

  --RS__zh-baseFontFamily:'\u65B9\u4F53', 'PingFang SC', '\u9ED1\u4F53', 'Heiti SC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK SC', sans-serif;

  --RS__th-lineHeightCompensation:1.067;

  --RS__th-baseFontFamily:Thonburi, 'Leelawadee UI', 'Cordia New', Roboto, Noto, 'Noto Sans Thai', sans-serif;

  --RS__te-baseFontFamily:'Kohinoor Telugu', 'Telugu Sangam MN', 'Nirmala UI', Gautami, Roboto, Noto, 'Noto Sans Telugu', sans-serif;

  --RS__ta-lineHeightCompensation:1.067;

  --RS__ta-baseFontFamily:'Tamil Sangam MN', 'Nirmala UI', Latha, Roboto, Noto, 'Noto Sans Tamil', sans-serif;

  --RS__si-lineHeightCompensation:1.167;

  --RS__si-baseFontFamily:'Sinhala Sangam MN', 'Nirmala UI', 'Iskoola Pota', Roboto, Noto, 'Noto Sans Sinhala', sans-serif;

  --RS__pa-lineHeightCompensation:1.1;

  --RS__pa-baseFontFamily:'Gurmukhi MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Gurmukhi', sans-serif;

  --RS__or-lineHeightCompensation:1.167;

  --RS__or-baseFontFamily:'Oriya Sangam MN', 'Nirmala UI', Kalinga, Roboto, Noto, 'Noto Sans Oriya', sans-serif;

  --RS__ml-lineHeightCompensation:1.067;

  --RS__ml-baseFontFamily:'Malayalam Sangam MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Malayalam', sans-serif;

  --RS__lo-baseFontFamily:'Lao Sangam MN', 'Leelawadee UI', 'Lao UI', Roboto, Noto, 'Noto Sans Lao', sans-serif;

  --RS__ko-lineHeightCompensation:1.167;

  --RS__ko-baseFontFamily:'Nanum Gothic', 'Apple SD Gothic Neo', 'Malgun Gothic', Roboto, Noto, 'Noto Sans CJK KR', sans-serif;

  --RS__kn-lineHeightCompensation:1.1;

  --RS__kn-baseFontFamily:'Kannada Sangam MN', 'Nirmala UI', Tunga, Roboto, Noto, 'Noto Sans Kannada', sans-serif;

  --RS__km-lineHeightCompensation:1.067;

  --RS__km-baseFontFamily:'Khmer Sangam MN', 'Leelawadee UI', 'Khmer UI', Roboto, Noto, 'Noto Sans Khmer', sans-serif;

  --RS__ja-lineHeightCompensation:1.167;

  --RS__ja-baseFontFamily:YuGothic, 'Hiragino Maru Gothic ProN', 'Hiragino Sans', 'Yu Gothic UI', 'Meiryo UI', 'MS Gothic', Roboto, Noto, 'Noto Sans CJK JP', sans-serif;

  --RS__iu-baseFontFamily:'Euphemia UCAS', Euphemia, Roboto, Noto, 'Noto Sans Canadian Aboriginal', sans-serif;

  --RS__hy-baseFontFamily:Mshtakan, Sylfaen, Roboto, Noto, 'Noto Serif Armenian', serif;

  --RS__hi-lineHeightCompensation:1.1;

  --RS__hi-baseFontFamily:'Kohinoor Devanagari', 'Devanagari Sangam MN', Kokila, 'Nirmala UI', Roboto, Noto, 'Noto Sans Devanagari', sans-serif;

  --RS__he-lineHeightCompensation:1.1;

  --RS__he-baseFontFamily:'New Peninim MT', 'Arial Hebrew', Gisha, 'Times New Roman', Roboto, Noto, 'Noto Sans Hebrew', sans-serif;

  --RS__gu-lineHeightCompensation:1.167;

  --RS__gu-baseFontFamily:'Gujarati Sangam MN', 'Nirmala UI', Shruti, Roboto, Noto, 'Noto Sans Gujarati', sans-serif;

  --RS__fa-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__chr-lineHeightCompensation:1.167;

  --RS__chr-baseFontFamily:'Plantagenet Cherokee', Roboto, Noto, 'Noto Sans Cherokee';

  --RS__bo-baseFontFamily:Kailasa, 'Microsoft Himalaya', Roboto, Noto, 'Noto Sans Tibetan', sans-serif;

  --RS__bn-lineHeightCompensation:1.067;

  --RS__bn-baseFontFamily:'Kohinoor Bangla', 'Bangla Sangam MN', Vrinda, Roboto, Noto, 'Noto Sans Bengali', sans-serif;

  --RS__ar-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__am-lineHeightCompensation:1.167;

  --RS__am-baseFontFamily:Kefa, Nyala, Roboto, Noto, 'Noto Sans Ethiopic', serif;

  --RS__latin-lineHeightCompensation:1;

  --RS__latin-baseFontFamily:var(--RS__oldStyleTf);
  --RS__baseFontFamily:var(--RS__latin-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__latin-lineHeightCompensation);
  --RS__baseLineHeight:calc(1.5 * var(--RS__lineHeightCompensation));

  --RS__selectionTextColor:inherit;

  --RS__selectionBackgroundColor:#b4d8fe;

  --RS__visitedColor:#551A8B;

  --RS__linkColor:#0000EE;

  --RS__textColor:#121212;

  --RS__backgroundColor:#FFFFFF;
  color:var(--RS__textColor) !important;

  background-color:var(--RS__backgroundColor) !important;
}

::-moz-selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

::selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

html{
  font-family:var(--RS__baseFontFamily);
  line-height:1.6;
  line-height:var(--RS__baseLineHeight);
  text-rendering:optimizelegibility;
}

h1, h2, h3{
  line-height:normal;
}

:lang(ja),
:lang(zh),
:lang(ko){
  word-wrap:break-word;
  -webkit-line-break:strict;
  -epub-line-break:strict;
  line-break:strict;
}

math{
  font-family:"Latin Modern Math", "STIX Two Math", "XITS Math", "STIX Math", "Libertinus Math", "TeX Gyre Termes Math", "TeX Gyre Bonum Math", "TeX Gyre Schola", "DejaVu Math TeX Gyre", "TeX Gyre Pagella Math", "Asana Math", "Cambria Math", "Lucida Bright Math", "Minion Math", STIXGeneral, STIXSizeOneSym, Symbol, "Times New Roman", serif;
}

:lang(am){
  --RS__baseFontFamily:var(--RS__am-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__am-lineHeightCompensation);
}

:lang(ar){
  --RS__baseFontFamily:var(--RS__ar-baseFontFamily);
}

:lang(bn){
  --RS__baseFontFamily:var(--RS__bn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__bn-lineHeightCompensation);
}

:lang(bo){
  --RS__baseFontFamily:var(--RS__bo-baseFontFamily);
}

:lang(chr){
  --RS__baseFontFamily:var(--RS__chr-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__chr-lineHeightCompensation);
}

:lang(fa){
  --RS__baseFontFamily:var(--RS__fa-baseFontFamily);
}

:lang(gu){
  --RS__baseFontFamily:var(--RS__gu-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__gu-lineHeightCompensation);
}

:lang(he){
  --RS__baseFontFamily:var(--RS__he-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__he-lineHeightCompensation);
}

:lang(hi){
  --RS__baseFontFamily:var(--RS__hi-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__hi-lineHeightCompensation);
}

:lang(hy){
  --RS__baseFontFamily:var(--RS__hy-baseFontFamily);
}

:lang(iu){
  --RS__baseFontFamily:var(--RS__iu-baseFontFamily);
}

:lang(ja){
  --RS__baseFontFamily:var(--RS__ja-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ja-lineHeightCompensation);
}

:lang(km){
  --RS__baseFontFamily:var(--RS__km-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__km-lineHeightCompensation);
}

:lang(kn){
  --RS__baseFontFamily:var(--RS__kn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__kn-lineHeightCompensation);
}

:lang(ko){
  --RS__baseFontFamily:var(--RS__ko-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ko-lineHeightCompensation);
}

:lang(lo){
  --RS__baseFontFamily:var(--RS__lo-baseFontFamily);
}

:lang(ml){
  --RS__baseFontFamily:var(--RS__ml-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ml-lineHeightCompensation);
}

:lang(or){
  --RS__baseFontFamily:var(--RS__or-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__or-lineHeightCompensation);
}

:lang(pa){
  --RS__baseFontFamily:var(--RS__pa-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__pa-lineHeightCompensation);
}

:lang(si){
  --RS__baseFontFamily:var(--RS__si-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__si-lineHeightCompensation);
}

:lang(ta){
  --RS__baseFontFamily:var(--RS__ta-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ta-lineHeightCompensation);
}

:lang(te){
  --RS__baseFontFamily:var(--RS__te-baseFontFamily);
}

:lang(th){
  --RS__baseFontFamily:var(--RS__th-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__th-lineHeightCompensation);
}

:lang(zh){
  --RS__baseFontFamily:var(--RS__zh-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-lineHeightCompensation);
}

:lang(zh-Hant){
  --RS__baseFontFamily:var(--RS__zh-Hant-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-Hant-lineHeightCompensation);
}

:lang(zh-TW){
  --RS__baseFontFamily:var(--RS__zh-TW-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-TW-lineHeightCompensation);
}

:lang(zh-HK){
  --RS__baseFontFamily:var(--RS__zh-HK-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-HK-lineHeightCompensation);
}

body{
  widows:2;
  orphans:2;
}

figcaption, th, td{
  widows:1;
  orphans:1;
}

h2, h3, h4, h5, h6, dt,
hr, caption{
  -webkit-column-break-after:avoid;
  page-break-after:avoid;
  break-after:avoid;
}

h1, h2, h3, h4, h5, h6, dt,
figure, tr{
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

body{
  -webkit-hyphenate-character:"\\002D";
  -moz-hyphenate-character:"\\002D";
  -ms-hyphenate-character:"\\002D";
  hyphenate-character:"\\002D";
  -webkit-hyphenate-limit-lines:3;
  -ms-hyphenate-limit-lines:3;
  hyphenate-limit-lines:3;
}

h1, h2, h3, h4, h5, h6, dt,
figcaption, pre, caption, address,
center, code, var{
  -ms-hyphens:none;
  -moz-hyphens:none;
  -webkit-hyphens:none;
  -epub-hyphens:none;
  hyphens:none;
}

body{
  font-variant-numeric:oldstyle-nums proportional-nums;
}

:lang(ja) body,
:lang(zh) body,
:lang(ko) body{
  font-variant-numeric:lining-nums proportional-nums;
}

h1, h2, h3, h4, h5, h6, dt{
  font-variant-numeric:lining-nums proportional-nums;
}

table{
  font-variant-numeric:lining-nums tabular-nums;
}

code, var{
  font-variant-ligatures:none;
  font-variant-numeric:lining-nums tabular-nums slashed-zero;
}

rt{
  font-variant-east-asian:ruby;
}

:lang(ar){
  font-variant-ligatures:common-ligatures;
}

:lang(ko){
  font-kerning:normal;
}

hr{
  color:inherit;
  border-color:currentcolor;
}

table, th, td{
  border-color:currentcolor;
}

figure, blockquote{
  margin:1em 5%;
}

ul, ol{
  padding-left:5%;
}

dd{
  margin-left:5%;
}

pre{
  white-space:pre-wrap;
  -ms-tab-size:2;
  -moz-tab-size:2;
  -webkit-tab-size:2;
  tab-size:2;
}

abbr[title], acronym[title]{
  text-decoration:dotted underline;
}

nobr wbr{
  white-space:normal;
}

ruby > rt, ruby > rp{
  -webkit-user-select:none;
  -moz-user-select:none;
  -ms-user-select:none;
  user-select:none;
}

*:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)),
*:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)),
*:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)),
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) cite, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) dfn, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) em, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) i,
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) cite, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) dfn, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) em, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) i,
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) cite, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) dfn, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) em, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) i{
  font-style:normal;
}

:lang(ja) a,
:lang(zh) a,
:lang(ko) a{
  text-decoration:none;
}

:root{
  --RS__maxMediaWidth:100%;
  --RS__maxMediaHeight:95vh;
  --RS__boxSizingMedia:border-box;
  --RS__boxSizingTable:border-box;
}

a, a span, span a, h1, h2, h3, h4, h5, h6{
  word-wrap:break-word;
}

div{
  max-width:var(--RS__maxMediaWidth);
}

img, svg|svg, video{
  object-fit:contain;

  width:auto;
  height:auto;
  max-width:var(--RS__maxMediaWidth);
  max-height:var(--RS__maxMediaHeight) !important;
  box-sizing:var(--RS__boxSizingMedia);
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

@supports (zoom: 1) and (not ((-webkit-column-axis: horizontal) and (-webkit-column-progression: normal))){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] img,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] svg|svg,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] video{
    zoom:calc(100% / var(--USER__fontSize));
  }
}

audio{
    max-width:100%;
    -webkit-column-break-inside:avoid;
    page-break-inside:avoid;
    break-inside:avoid;
  }

table{
  max-width:var(--RS__maxMediaWidth);
  box-sizing:var(--RS__boxSizingTable);
}`});var Yr={};nt(Yr,{default:()=>Va});var Va,qr=ot(()=>{Va=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

  --RS__compFontFamily:var(--RS__baseFontFamily);
  --RS__codeFontFamily:var(--RS__monospaceTf);

  --RS__typeScale:1.125;
  --RS__baseFontSize:87.5%;

  --RS__flowSpacing:1.5rem;
  --RS__paraSpacing:0;
  --RS__paraIndent:1em;

  --RS__linkColor:#0000EE;
  --RS__visitedColor:#551A8B;

  --RS__primaryColor:;
  --RS__secondaryColor:;
}

:root:lang(zh){
  --RS__paraIndent:2em;
}

:root{
  quotes:"\\201c" "\\201d" "\\2018" "\\2019";
}

body{
  font-size:var(--RS__baseFontSize);
  text-align:justify;
  text-justify:inter-character;
}

h1, h2, h3, h4, h5, h6{
  font-family:var(--RS__baseFontFamily);
  text-align:left;
  text-align:start;
}

blockquote, figure, p, pre,
aside, footer, form, hr{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

p{
  margin-top:var(--RS__paraSpacing);
  margin-bottom:var(--RS__paraSpacing);
  text-indent:var(--RS__paraIndent);
}

pre{
  font-family:var(--RS__codeFontFamily);
}

code, kbd, samp, tt{
  font-family:var(--RS__codeFontFamily);
}

sub, sup{
  position:relative;
  font-size:67.5%;
  line-height:1;
}

sub{
  bottom:-0.2ex;
}

sup{
  bottom:0;
}

em{
  -webkit-text-emphasis:dot;
  -epub-text-emphasis:dot;
  text-emphasis:dot;
}

:link{
  color:var(--RS__linkColor);
}

:visited{
  color:var(--RS__visitedColor);
}

h1{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:calc(var(--RS__flowSpacing) * 2);
  font-size:calc(((1em * var(--RS__typeScale)) * var(--RS__typeScale)) * var(--RS__typeScale));
  text-align:center;
}

h2{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc((1em * var(--RS__typeScale)) * var(--RS__typeScale));
  text-align:center;
}

h3{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc(1em * var(--RS__typeScale));
  text-align:center;
}

h4{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-family:var(--RS__compFontFamily);
  font-size:1em;
}

h5{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-family:var(--RS__compFontFamily);
  font-size:smaller;
}

h6{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:0;
  font-family:var(--RS__compFontFamily);
  font-size:smaller;
  font-weight:normal;
}

dl, ol, ul{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

table{
  margin:var(--RS__flowSpacing) 0;
  border:1px solid currentcolor;
  border-collapse:collapse;
  empty-cells:show;
}

thead, tbody, tfoot, table > tr{
  vertical-align:top;
}

th{
  text-align:left;
}

th, td{
  padding:4px;
  border:1px solid currentcolor;
}`});var Jr={};nt(Jr,{default:()=>Xa});var Xa,Zr=ot(()=>{Xa=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

  --RS__viewportWidth:100%;

  --RS__pageGutter:0;

  --RS__defaultLineLength:100%;

  --RS__colGap:0;

  --RS__colCount:1;

  --RS__colWidth:100vw;
}

@page{
  margin:0 !important;
}

:root{
  position:relative;

  -webkit-column-width:var(--RS__colWidth);
  -moz-column-width:var(--RS__colWidth);
  column-width:var(--RS__colWidth);
  -webkit-column-count:var(--RS__colCount);
  -moz-column-count:var(--RS__colCount);
  column-count:var(--RS__colCount);

  -webkit-column-gap:var(--RS__colGap);
  -moz-column-gap:var(--RS__colGap);
  column-gap:var(--RS__colGap);
  -moz-column-fill:auto;
  column-fill:auto;
  width:var(--RS__viewportWidth);
  height:100vh;
  max-width:var(--RS__viewportWidth);
  max-height:100vh;
  min-width:var(--RS__viewportWidth);
  min-height:100vh;
  padding:0 !important;
  margin:0 !important;
  font-size:1rem !important;
  box-sizing:border-box;
  -webkit-touch-callout:none;
}

body{
  width:100%;
  max-width:var(--RS__defaultLineLength) !important;
  margin:0 auto !important;
  box-sizing:border-box;
}

:root:not([style*="readium-scroll-on"]) body{
  padding:0 var(--RS__pageGutter) !important;
}

:root:not([style*="readium-noOverflow-on"]) body{
  overflow:hidden;
}

@supports (overflow: clip){

   :root:not([style*="readium-noOverflow-on"]){
      overflow:clip;
   }

   :root:not([style*="readium-noOverflow-on"]) body{
      overflow:clip;
      overflow-clip-margin:content-box;
   }
}

:root[style*="readium-scroll-on"]{
  -webkit-columns:auto auto !important;
  -moz-columns:auto auto !important;
  columns:auto auto !important;
  width:auto !important;
  height:auto !important;
  max-width:none !important;
  max-height:none !important;
  min-width:0 !important;
  min-height:0 !important;
}

:root[style*="readium-scroll-on"] body{
  max-width:var(--RS__defaultLineLength) !important;
  box-sizing:border-box !important;
}

:root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
  overflow:auto;
}

@supports (overflow: clip){

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]){
     overflow:auto;
  }

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
     overflow:clip;
  }
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingTop"] body{
  padding-top:var(--RS__scrollPaddingTop) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingBottom"] body{
  padding-bottom:var(--RS__scrollPaddingBottom) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingLeft"] body{
  padding-left:var(--RS__scrollPaddingLeft) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingRight"] body{
  padding-right:var(--RS__scrollPaddingRight) !important;
}

:root[style*="--USER__backgroundColor"]{
  background-color:var(--USER__backgroundColor) !important;
}

:root[style*="--USER__backgroundColor"] *{
  background-color:transparent !important;
}

:root[style*="--USER__textColor"]{
  color:var(--USER__textColor) !important;
}

:root[style*="--USER__textColor"] *:not(a){
  color:inherit !important;
  background-color:transparent !important;
  border-color:currentcolor !important;
}

:root[style*="--USER__textColor"] svg text{
  fill:currentcolor !important;
  stroke:none !important;
}

:root[style*="--USER__linkColor"] a:link,
:root[style*="--USER__linkColor"] a:link *{
  color:var(--USER__linkColor) !important;
}

:root[style*="--USER__visitedColor"] a:visited,
:root[style*="--USER__visitedColor"] a:visited *{
  color:var(--USER__visitedColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::-moz-selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__colCount"]{
  -webkit-column-count:var(--USER__colCount);
  -moz-column-count:var(--USER__colCount);
  column-count:var(--USER__colCount);

  --RS__colWidth:auto;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"]{
  -webkit-column-count:1;
  -moz-column-count:1;
  column-count:1;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"],
:root[style*="--USER__colCount: 1"],
:root[style*="--USER__colCount:1"]{
  --RS__colWidth:100vw;
}

:root[style*="--USER__lineLength"] body{
    max-width:var(--USER__lineLength) !important;
  }

:root[style*="--USER__fontFamily"]{
  font-family:var(--USER__fontFamily) !important;
}

:root[style*="--USER__fontFamily"] *{
  font-family:revert !important;
}

:root:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] body{
  zoom:var(--USER__fontSize) !important;
}

:root:not([style*="readium-deprecatedFontSize-on"])[style*="readium-iOSPatch-on"][style*="--USER__fontSize"] body{
  -webkit-text-size-adjust:var(--USER__fontSize) !important;
}

@supports selector(figure:has(> img)){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> img),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> video),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> svg),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> canvas),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> iframe),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> audio),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> img:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> video:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> svg:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> canvas:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> iframe:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> audio:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] table{
    zoom:calc(100% / var(--USER__fontSize)) !important;
  }

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figcaption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] caption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] td,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] th{
    zoom:var(--USER__fontSize) !important;
  }
}

@supports not (zoom: 1){

  :root[style*="--USER__fontSize"]{
    font-size:var(--USER__fontSize) !important;
  }
}

:root[style*="readium-deprecatedFontSize-on"][style*="--USER__fontSize"]{
  font-size:var(--USER__fontSize) !important;
}

:root[style*="--USER__lineHeight"]{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__lineHeight"] body,
:root[style*="--USER__lineHeight"] p,
:root[style*="--USER__lineHeight"] li,
:root[style*="--USER__lineHeight"] div{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__paraSpacing"] p{
  margin-top:var(--USER__paraSpacing) !important;
  margin-bottom:var(--USER__paraSpacing) !important;
}

:root[style*="--USER__fontWeight"] body{
  font-weight:var(--USER__fontWeight) !important;
}

:root[style*="--USER__fontWeight"] b,
:root[style*="--USER__fontWeight"] strong{
  font-weight:bolder;
}

:root[style*="--USER__fontWidth"] body{
  font-stretch:var(--USER__fontWidth) !important;
}

:root[style*="--USER__fontOpticalSizing"] body{
  font-optical-sizing:var(--USER__fontOpticalSizing) !important;
}

:root[style*="--USER__letterSpacing"] h1,
:root[style*="--USER__letterSpacing"] h2,
:root[style*="--USER__letterSpacing"] h3,
:root[style*="--USER__letterSpacing"] h4,
:root[style*="--USER__letterSpacing"] h5,
:root[style*="--USER__letterSpacing"] h6,
:root[style*="--USER__letterSpacing"] p,
:root[style*="--USER__letterSpacing"] li,
:root[style*="--USER__letterSpacing"] div,
:root[style*="--USER__letterSpacing"] dt,
:root[style*="--USER__letterSpacing"] dd{
  letter-spacing:var(--USER__letterSpacing) !important;
  font-variant:none !important;
}

:root[style*="readium-noRuby-on"] body rt,
:root[style*="readium-noRuby-on"] body rp{
  display:none;
}

:root[style*="readium-blend-on"] svg,
:root[style*="readium-blend-on"] img{
  background-color:transparent !important;
  mix-blend-mode:multiply !important;
}

:root[style*="--USER__darkenImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) !important;
  filter:brightness(var(--USER__darkenImages)) !important;
}

:root[style*="readium-darken-on"] img{
  -webkit-filter:brightness(80%) !important;
  filter:brightness(80%) !important;
}

:root[style*="--USER__invertImages"] img{
  -webkit-filter:invert(var(--USER__invertImages)) !important;
  filter:invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-invert-on"] img{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="--USER__darkenImages"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
  filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-darken-on"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(80%) invert(var(--USER__invertImages)) !important;
  filter:brightness(80%) invert(var(--USER__invertImages)) !important;
}

:root[style*="--USER__darkenImages"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
  filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
}

:root[style*="readium-darken-on"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(80%) invert(100%) !important;
  filter:brightness(80%) invert(100%) !important;
}

:root[style*="--USER__invertGaiji"] img[class*="gaiji"]{
  -webkit-filter:invert(var(--USER__invertGaiji)) !important;
  filter:invert(var(--USER__invertGaiji)) !important;
}

:root[style*="readium-invertGaiji-on"] img[class*="gaiji"]{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="readium-normalize-on"]{
  --USER__typeScale:1.2;
}

:root[style*="readium-normalize-on"] p,
:root[style*="readium-normalize-on"] li,
:root[style*="readium-normalize-on"] div,
:root[style*="readium-normalize-on"] pre,
:root[style*="readium-normalize-on"] dd{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] h1{
  font-size:1.75rem !important;
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h2{
  font-size:1.5rem !important;
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h3{
  font-size:1.25rem !important;
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h4,
:root[style*="readium-normalize-on"] h5,
:root[style*="readium-normalize-on"] h6{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] small{
  font-size:smaller !important;
}

:root[style*="readium-normalize-on"] sub,
:root[style*="readium-normalize-on"] sup{
  font-size:67.5% !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h1{
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h2{
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h3{
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-iPadOSPatch-on"] body{
  -webkit-text-size-adjust:none;
}

:root[style*="readium-iPadOSPatch-on"] p, 
:root[style*="readium-iPadOSPatch-on"] h1, 
:root[style*="readium-iPadOSPatch-on"] h2, 
:root[style*="readium-iPadOSPatch-on"] h3, 
:root[style*="readium-iPadOSPatch-on"] h4, 
:root[style*="readium-iPadOSPatch-on"] h5, 
:root[style*="readium-iPadOSPatch-on"] h6, 
:root[style*="readium-iPadOSPatch-on"] li, 
:root[style*="readium-iPadOSPatch-on"] th, 
:root[style*="readium-iPadOSPatch-on"] td, 
:root[style*="readium-iPadOSPatch-on"] dt, 
:root[style*="readium-iPadOSPatch-on"] dd, 
:root[style*="readium-iPadOSPatch-on"] pre, 
:root[style*="readium-iPadOSPatch-on"] address, 
:root[style*="readium-iPadOSPatch-on"] details, 
:root[style*="readium-iPadOSPatch-on"] summary,
:root[style*="readium-iPadOSPatch-on"] figcaption,
:root[style*="readium-iPadOSPatch-on"] div:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)),
:root[style*="readium-iPadOSPatch-on"] aside:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)){
  -webkit-text-zoom:reset;
}

:root[style*="readium-iPadOSPatch-on"] abbr, 
:root[style*="readium-iPadOSPatch-on"] b, 
:root[style*="readium-iPadOSPatch-on"] bdi, 
:root[style*="readium-iPadOSPatch-on"] bdo, 
:root[style*="readium-iPadOSPatch-on"] cite, 
:root[style*="readium-iPadOSPatch-on"] code, 
:root[style*="readium-iPadOSPatch-on"] dfn, 
:root[style*="readium-iPadOSPatch-on"] em, 
:root[style*="readium-iPadOSPatch-on"] i, 
:root[style*="readium-iPadOSPatch-on"] kbd, 
:root[style*="readium-iPadOSPatch-on"] mark, 
:root[style*="readium-iPadOSPatch-on"] q, 
:root[style*="readium-iPadOSPatch-on"] rp, 
:root[style*="readium-iPadOSPatch-on"] rt, 
:root[style*="readium-iPadOSPatch-on"] ruby, 
:root[style*="readium-iPadOSPatch-on"] s, 
:root[style*="readium-iPadOSPatch-on"] samp, 
:root[style*="readium-iPadOSPatch-on"] small, 
:root[style*="readium-iPadOSPatch-on"] span, 
:root[style*="readium-iPadOSPatch-on"] strong, 
:root[style*="readium-iPadOSPatch-on"] sub, 
:root[style*="readium-iPadOSPatch-on"] sup, 
:root[style*="readium-iPadOSPatch-on"] time, 
:root[style*="readium-iPadOSPatch-on"] u, 
:root[style*="readium-iPadOSPatch-on"] var{
  -webkit-text-zoom:normal;
}

:root[style*="readium-iPadOSPatch-on"] p:not(:has(b, cite, em, i, q, s, small, span, strong)):first-line{
  -webkit-text-zoom:normal;
}`});var Qr={};nt(Qr,{default:()=>$a});var $a,ts=ot(()=>{$a=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

@-ms-viewport{
  width:device-width;
}

@viewport{
  width:device-width;
  zoom:1;
}

:root{

  --RS__sans-serif-ja-v:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDGothic', 'Yu Gothic', '\uFF2D\uFF33\u30B4\u30B7\u30C3\u30AF', 'MS Gothic', sans-serif;

  --RS__serif-ja-v:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDMincho', 'Yu Mincho', '\uFF2D\uFF33\u660E\u671D', 'MS Mincho', serif;

  --RS__sans-serif-ja:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDPGothic', 'Yu Gothic', '\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF', 'MS PGothic', sans-serif;

  --RS__serif-ja:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDPMincho', 'Yu Mincho', '\uFF2D\uFF33 \uFF30\u660E\u671D', 'MS PMincho', serif;

  --RS__monospaceTf:ui-monospace, 'Andale Mono', 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;

  --RS__humanistTf:Seravek, Calibri, 'Gill Sans Nova', Roboto, Ubuntu, 'DejaVu Sans', source-sans-pro, sans-serif;

  --RS__sansTf:-ui-sans-serif, -apple-system, system-ui, BlinkMacSystemFont, 'Segoe UI Variable', 'Segoe UI', Inter, Roboto, 'Helvetica Neue', 'Arial Nova', 'Liberation Sans', Arial, sans-serif;

  --RS__modernTf:Athelas, Constantia, Charter, 'Bitstream Charter', Cambria, 'Georgia Pro', Georgia, serif;

  --RS__oldStyleTf:'Iowan Old Style', Sitka, 'Sitka Text', Palatino, 'Book Antiqua', 'URW Palladio L', P052, serif;

  --RS__zh-HK-lineHeightCompensation:1.167;

  --RS__zh-HK-baseFontFamily:'\u65B9\u9AD4', 'PingFang HK', '\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-TW-lineHeightCompensation:1.167;

  --RS__zh-TW-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-Hant-lineHeightCompensation:1.167;

  --RS__zh-Hant-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-lineHeightCompensation:1.167;

  --RS__zh-baseFontFamily:'\u65B9\u4F53', 'PingFang SC', '\u9ED1\u4F53', 'Heiti SC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK SC', sans-serif;

  --RS__th-lineHeightCompensation:1.067;

  --RS__th-baseFontFamily:Thonburi, 'Leelawadee UI', 'Cordia New', Roboto, Noto, 'Noto Sans Thai', sans-serif;

  --RS__te-baseFontFamily:'Kohinoor Telugu', 'Telugu Sangam MN', 'Nirmala UI', Gautami, Roboto, Noto, 'Noto Sans Telugu', sans-serif;

  --RS__ta-lineHeightCompensation:1.067;

  --RS__ta-baseFontFamily:'Tamil Sangam MN', 'Nirmala UI', Latha, Roboto, Noto, 'Noto Sans Tamil', sans-serif;

  --RS__si-lineHeightCompensation:1.167;

  --RS__si-baseFontFamily:'Sinhala Sangam MN', 'Nirmala UI', 'Iskoola Pota', Roboto, Noto, 'Noto Sans Sinhala', sans-serif;

  --RS__pa-lineHeightCompensation:1.1;

  --RS__pa-baseFontFamily:'Gurmukhi MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Gurmukhi', sans-serif;

  --RS__or-lineHeightCompensation:1.167;

  --RS__or-baseFontFamily:'Oriya Sangam MN', 'Nirmala UI', Kalinga, Roboto, Noto, 'Noto Sans Oriya', sans-serif;

  --RS__ml-lineHeightCompensation:1.067;

  --RS__ml-baseFontFamily:'Malayalam Sangam MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Malayalam', sans-serif;

  --RS__lo-baseFontFamily:'Lao Sangam MN', 'Leelawadee UI', 'Lao UI', Roboto, Noto, 'Noto Sans Lao', sans-serif;

  --RS__ko-lineHeightCompensation:1.167;

  --RS__ko-baseFontFamily:'Nanum Gothic', 'Apple SD Gothic Neo', 'Malgun Gothic', Roboto, Noto, 'Noto Sans CJK KR', sans-serif;

  --RS__kn-lineHeightCompensation:1.1;

  --RS__kn-baseFontFamily:'Kannada Sangam MN', 'Nirmala UI', Tunga, Roboto, Noto, 'Noto Sans Kannada', sans-serif;

  --RS__km-lineHeightCompensation:1.067;

  --RS__km-baseFontFamily:'Khmer Sangam MN', 'Leelawadee UI', 'Khmer UI', Roboto, Noto, 'Noto Sans Khmer', sans-serif;

  --RS__ja-lineHeightCompensation:1.167;

  --RS__ja-baseFontFamily:YuGothic, 'Hiragino Maru Gothic ProN', 'Hiragino Sans', 'Yu Gothic UI', 'Meiryo UI', 'MS Gothic', Roboto, Noto, 'Noto Sans CJK JP', sans-serif;

  --RS__iu-baseFontFamily:'Euphemia UCAS', Euphemia, Roboto, Noto, 'Noto Sans Canadian Aboriginal', sans-serif;

  --RS__hy-baseFontFamily:Mshtakan, Sylfaen, Roboto, Noto, 'Noto Serif Armenian', serif;

  --RS__hi-lineHeightCompensation:1.1;

  --RS__hi-baseFontFamily:'Kohinoor Devanagari', 'Devanagari Sangam MN', Kokila, 'Nirmala UI', Roboto, Noto, 'Noto Sans Devanagari', sans-serif;

  --RS__he-lineHeightCompensation:1.1;

  --RS__he-baseFontFamily:'New Peninim MT', 'Arial Hebrew', Gisha, 'Times New Roman', Roboto, Noto, 'Noto Sans Hebrew', sans-serif;

  --RS__gu-lineHeightCompensation:1.167;

  --RS__gu-baseFontFamily:'Gujarati Sangam MN', 'Nirmala UI', Shruti, Roboto, Noto, 'Noto Sans Gujarati', sans-serif;

  --RS__fa-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__chr-lineHeightCompensation:1.167;

  --RS__chr-baseFontFamily:'Plantagenet Cherokee', Roboto, Noto, 'Noto Sans Cherokee';

  --RS__bo-baseFontFamily:Kailasa, 'Microsoft Himalaya', Roboto, Noto, 'Noto Sans Tibetan', sans-serif;

  --RS__bn-lineHeightCompensation:1.067;

  --RS__bn-baseFontFamily:'Kohinoor Bangla', 'Bangla Sangam MN', Vrinda, Roboto, Noto, 'Noto Sans Bengali', sans-serif;

  --RS__ar-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__am-lineHeightCompensation:1.167;

  --RS__am-baseFontFamily:Kefa, Nyala, Roboto, Noto, 'Noto Sans Ethiopic', serif;

  --RS__latin-lineHeightCompensation:1;

  --RS__latin-baseFontFamily:var(--RS__oldStyleTf);
  --RS__baseFontFamily:var(--RS__latin-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__latin-lineHeightCompensation);
  --RS__baseLineHeight:calc(1.5 * var(--RS__lineHeightCompensation));

  --RS__selectionTextColor:inherit;

  --RS__selectionBackgroundColor:#b4d8fe;

  --RS__visitedColor:#551A8B;

  --RS__linkColor:#0000EE;

  --RS__textColor:#121212;

  --RS__backgroundColor:#FFFFFF;
  color:var(--RS__textColor) !important;

  background-color:var(--RS__backgroundColor) !important;
}

::-moz-selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

::selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

html{
  font-family:var(--RS__baseFontFamily);
  line-height:1.6;
  line-height:var(--RS__baseLineHeight);
  text-rendering:optimizelegibility;
}

h1, h2, h3{
  line-height:normal;
}

:lang(ja),
:lang(zh),
:lang(ko){
  word-wrap:break-word;
  -webkit-line-break:strict;
  -epub-line-break:strict;
  line-break:strict;
}

math{
  font-family:"Latin Modern Math", "STIX Two Math", "XITS Math", "STIX Math", "Libertinus Math", "TeX Gyre Termes Math", "TeX Gyre Bonum Math", "TeX Gyre Schola", "DejaVu Math TeX Gyre", "TeX Gyre Pagella Math", "Asana Math", "Cambria Math", "Lucida Bright Math", "Minion Math", STIXGeneral, STIXSizeOneSym, Symbol, "Times New Roman", serif;
}

:lang(am){
  --RS__baseFontFamily:var(--RS__am-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__am-lineHeightCompensation);
}

:lang(ar){
  --RS__baseFontFamily:var(--RS__ar-baseFontFamily);
}

:lang(bn){
  --RS__baseFontFamily:var(--RS__bn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__bn-lineHeightCompensation);
}

:lang(bo){
  --RS__baseFontFamily:var(--RS__bo-baseFontFamily);
}

:lang(chr){
  --RS__baseFontFamily:var(--RS__chr-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__chr-lineHeightCompensation);
}

:lang(fa){
  --RS__baseFontFamily:var(--RS__fa-baseFontFamily);
}

:lang(gu){
  --RS__baseFontFamily:var(--RS__gu-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__gu-lineHeightCompensation);
}

:lang(he){
  --RS__baseFontFamily:var(--RS__he-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__he-lineHeightCompensation);
}

:lang(hi){
  --RS__baseFontFamily:var(--RS__hi-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__hi-lineHeightCompensation);
}

:lang(hy){
  --RS__baseFontFamily:var(--RS__hy-baseFontFamily);
}

:lang(iu){
  --RS__baseFontFamily:var(--RS__iu-baseFontFamily);
}

:lang(ja){
  --RS__baseFontFamily:var(--RS__ja-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ja-lineHeightCompensation);
}

:lang(km){
  --RS__baseFontFamily:var(--RS__km-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__km-lineHeightCompensation);
}

:lang(kn){
  --RS__baseFontFamily:var(--RS__kn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__kn-lineHeightCompensation);
}

:lang(ko){
  --RS__baseFontFamily:var(--RS__ko-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ko-lineHeightCompensation);
}

:lang(lo){
  --RS__baseFontFamily:var(--RS__lo-baseFontFamily);
}

:lang(ml){
  --RS__baseFontFamily:var(--RS__ml-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ml-lineHeightCompensation);
}

:lang(or){
  --RS__baseFontFamily:var(--RS__or-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__or-lineHeightCompensation);
}

:lang(pa){
  --RS__baseFontFamily:var(--RS__pa-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__pa-lineHeightCompensation);
}

:lang(si){
  --RS__baseFontFamily:var(--RS__si-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__si-lineHeightCompensation);
}

:lang(ta){
  --RS__baseFontFamily:var(--RS__ta-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ta-lineHeightCompensation);
}

:lang(te){
  --RS__baseFontFamily:var(--RS__te-baseFontFamily);
}

:lang(th){
  --RS__baseFontFamily:var(--RS__th-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__th-lineHeightCompensation);
}

:lang(zh){
  --RS__baseFontFamily:var(--RS__zh-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-lineHeightCompensation);
}

:lang(zh-Hant){
  --RS__baseFontFamily:var(--RS__zh-Hant-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-Hant-lineHeightCompensation);
}

:lang(zh-TW){
  --RS__baseFontFamily:var(--RS__zh-TW-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-TW-lineHeightCompensation);
}

:lang(zh-HK){
  --RS__baseFontFamily:var(--RS__zh-HK-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-HK-lineHeightCompensation);
}

body{
  widows:2;
  orphans:2;
}

figcaption, th, td{
  widows:1;
  orphans:1;
}

h2, h3, h4, h5, h6, dt,
hr, caption{
  -webkit-column-break-after:avoid;
  page-break-after:avoid;
  break-after:avoid;
}

h1, h2, h3, h4, h5, h6, dt,
figure, tr{
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

body{
  -webkit-hyphenate-character:"\\002D";
  -moz-hyphenate-character:"\\002D";
  -ms-hyphenate-character:"\\002D";
  hyphenate-character:"\\002D";
  -webkit-hyphenate-limit-lines:3;
  -ms-hyphenate-limit-lines:3;
  hyphenate-limit-lines:3;
}

h1, h2, h3, h4, h5, h6, dt,
figcaption, pre, caption, address,
center, code, var{
  -ms-hyphens:none;
  -moz-hyphens:none;
  -webkit-hyphens:none;
  -epub-hyphens:none;
  hyphens:none;
}

body{
  font-variant-numeric:oldstyle-nums proportional-nums;
}

:lang(ja) body,
:lang(zh) body,
:lang(ko) body{
  font-variant-numeric:lining-nums proportional-nums;
}

h1, h2, h3, h4, h5, h6, dt{
  font-variant-numeric:lining-nums proportional-nums;
}

table{
  font-variant-numeric:lining-nums tabular-nums;
}

code, var{
  font-variant-ligatures:none;
  font-variant-numeric:lining-nums tabular-nums slashed-zero;
}

rt{
  font-variant-east-asian:ruby;
}

:lang(ar){
  font-variant-ligatures:common-ligatures;
}

:lang(ko){
  font-kerning:normal;
}

hr{
  color:inherit;
  border-color:currentcolor;
}

table, th, td{
  border-color:currentcolor;
}

figure, blockquote{
  margin:1em 5%;
}

ul, ol{
  padding-left:5%;
}

dd{
  margin-left:5%;
}

pre{
  white-space:pre-wrap;
  -ms-tab-size:2;
  -moz-tab-size:2;
  -webkit-tab-size:2;
  tab-size:2;
}

abbr[title], acronym[title]{
  text-decoration:dotted underline;
}

nobr wbr{
  white-space:normal;
}

ruby > rt, ruby > rp{
  -webkit-user-select:none;
  -moz-user-select:none;
  -ms-user-select:none;
  user-select:none;
}

*:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)),
*:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)),
*:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)),
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) cite, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) dfn, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) em, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) i,
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) cite, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) dfn, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) em, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) i,
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) cite, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) dfn, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) em, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) i{
  font-style:normal;
}

:lang(ja) a,
:lang(zh) a,
:lang(ko) a{
  text-decoration:none;
}

:root{
  --RS__maxMediaWidth:100%;
  --RS__maxMediaHeight:100vw;
  --RS__boxSizingMedia:border-box;
  --RS__boxSizingTable:border-box;
}

a, a span, span a, h1, h2, h3, h4, h5, h6{
  word-wrap:break-word;
}

div{
  max-width:var(--RS__maxMediaHeight);
}

img, svg|svg, video{
  object-fit:contain;

  width:auto;
  height:auto;
  max-width:var(--RS__maxMediaHeight);
  max-height:var(--RS__maxMediaWidth) !important;
  box-sizing:var(--RS__boxSizingMedia);
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

@supports (zoom: 1) and (not ((-webkit-column-axis: horizontal) and (-webkit-column-progression: normal))){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] img,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] svg|svg,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] video,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div{
    zoom:calc(100% / var(--USER__fontSize));
  }
}

audio{
  max-width:100%;
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

table{
  max-height:var(--RS__maxMediaWidth);
  box-sizing:var(--RS__boxSizingTable);
}`});var es={};nt(es,{default:()=>Ka});var Ka,is=ot(()=>{Ka=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

  --RS__compFontFamily:var(--RS__baseFontFamily);
  --RS__codeFontFamily:var(--RS__monospaceTf);

  --RS__typeScale:1.125;
  --RS__baseFontSize:87.5%;

  --RS__flowSpacing:1.5rem;
  --RS__paraSpacing:0;
  --RS__paraIndent:1em;

  --RS__linkColor:#0000EE;
  --RS__visitedColor:#551A8B;

  --RS__primaryColor:;
  --RS__secondaryColor:;
}

:root:lang(zh){
  --RS__paraIndent:2em;
}

:lang("mn-Mong"){
  --RS__baseFontSize:100%;
}

body{
  font-size:var(--RS__baseFontSize);
  text-align:justify;
  text-justify:inter-character;
}

h1, h2, h3, h4, h5, h6{
  font-family:var(--RS__baseFontFamily);
  text-align:left;
  text-align:start;
}

blockquote, figure, p, pre,
aside, footer, form, hr{
  margin-right:var(--RS__flowSpacing);
  margin-left:var(--RS__flowSpacing);
}

p{
  margin-right:var(--RS__paraSpacing);
  margin-left:var(--RS__paraSpacing);
  text-indent:var(--RS__paraIndent);
}

pre{
  font-family:var(--RS__codeFontFamily);
}

code, kbd, samp, tt{
  font-family:var(--RS__codeFontFamily);
}

sub, sup{
  position:relative;
  font-size:67.5%;
  line-height:1;
}

sub{
  left:-0.2ex;
}

sup{
  right:0;
}

em{
  -webkit-text-emphasis:sesame;
  -epub-text-emphasis:sesame;
  text-emphasis:sesame;
}

:link{
  color:var(--RS__linkColor);
}

:visited{
  color:var(--RS__visitedColor);
}

h1{
  margin-right:calc(var(--RS__flowSpacing) * 2);
  margin-left:calc(var(--RS__flowSpacing) * 2);
  font-size:calc(((1em * var(--RS__typeScale)) * var(--RS__typeScale)) * var(--RS__typeScale));
  text-indent:2rem;
}

h2{
  margin-right:calc(var(--RS__flowSpacing) * 2);
  margin-left:var(--RS__flowSpacing);
  font-size:calc((1em * var(--RS__typeScale)) * var(--RS__typeScale));
  text-indent:3rem;
}

h3{
  margin-right:var(--RS__flowSpacing);
  margin-left:var(--RS__flowSpacing);
  font-size:calc(1em * var(--RS__typeScale));
  text-indent:4rem;
}

h4{
  margin-right:var(--RS__flowSpacing);
  margin-left:var(--RS__flowSpacing);
  font-family:var(--RS__compFontFamily);
  font-size:1em;
  text-indent:4rem;
}

h5{
  margin-right:var(--RS__flowSpacing);
  margin-left:var(--RS__flowSpacing);
  font-family:var(--RS__compFontFamily);
  font-size:smaller;
  text-indent:4rem;
}

h6{
  margin-right:var(--RS__flowSpacing);
  margin-left:0;
  font-family:var(--RS__compFontFamily);
  font-size:smaller;
  font-weight:normal;
  text-indent:4rem;
}

dl, ol, ul{
  margin-right:var(--RS__flowSpacing);
  margin-left:var(--RS__flowSpacing);
}

table{
  margin:0 var(--RS__flowSpacing);
  border:1px solid currentcolor;
  border-collapse:collapse;
  empty-cells:show;
}

thead, tbody, tfoot, table > tr{
  vertical-align:top;
}

th{
  text-align:left;
}

th, td{
  padding:4px;
  border:1px solid currentcolor;
}`});var os={};nt(os,{default:()=>Ya});var Ya,ns=ot(()=>{Ya=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

   --RS__viewportWidth:100%;

   --RS__pageGutter:0;

   --RS__defaultLineLength:100%;

   --RS__colGap:0;

   --RS__colCount:1;

   --RS__colWidth:100vw;
}

@page{
  margin:0 !important;
}

:root{
  position:relative;

  -webkit-column-width:var(--RS__colWidth);
  -moz-column-width:var(--RS__colWidth);
  column-width:var(--RS__colWidth);
  -webkit-column-count:var(--RS__colCount);
  -moz-column-count:var(--RS__colCount);
  column-count:var(--RS__colCount);

  -webkit-column-gap:var(--RS__colGap);
  -moz-column-gap:var(--RS__colGap);
  column-gap:var(--RS__colGap);
  -moz-column-fill:auto;
  column-fill:auto;
  width:100%;
  height:100vh;
  max-width:100%;
  max-height:100vh;
  min-width:100%;
  min-height:100vh;
  padding:0 !important;
  margin:0 !important;
  font-size:1rem !important;
  box-sizing:border-box;

  hanging-punctuation:last allow-end;
  -webkit-touch-callout:none;
  -ms-writing-mode:tb-rl;
  -webkit-writing-mode:vertical-rl;
  writing-mode:vertical-rl;
}

:root:lang(mn-Mong){
  -ms-writing-mode:tb;
  -webkit-writing-mode:vertical-lr;
  writing-mode:vertical-lr;
}

body{
  width:100%;
  max-height:var(--RS__defaultLineLength) !important;
  margin:auto 0 !important;
  box-sizing:border-box;
}

:root:not([style*="readium-scroll-on"]) body{
  padding:var(--RS__pageGutter) 0 !important;
}

:root:not([style*="readium-noOverflow-on"]) body{
  overflow:hidden;
}

@supports (overflow: clip){

  :root:not([style*="readium-noOverflow-on"]){
     overflow:clip;
  }

  :root:not([style*="readium-noOverflow-on"]) body{
     overflow:clip;
     overflow-clip-margin:content-box;
  }
}

:root[style*="readium-scroll-on"],
:root[style*="readium-noVerticalPagination-on"]{
  -webkit-columns:auto auto !important;
  -moz-columns:auto auto !important;
  columns:auto auto !important;
  width:auto !important;
  max-width:none !important;
  max-height:100vh !important;
  min-width:0 !important;
}

:root[style*="readium-scroll-on"] body,
:root[style*="readium-noVerticalPagination-on"] body{
  max-width:var(--RS__defaultLineLength) !important;
  box-sizing:border-box !important;
}

@supports (overflow: clip){

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]){
     overflow:auto;
  }

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
     overflow:clip;
  }
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingTop"] body{
  padding-top:var(--RS__scrollPaddingTop) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingBottom"] body{
  padding-bottom:var(--RS__scrollPaddingBottom) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingLeft"] body{
  padding-left:var(--RS__scrollPaddingLeft) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingRight"] body{
  padding-right:var(--RS__scrollPaddingRight) !important;
}

:root[style*="--USER__backgroundColor"]{
  background-color:var(--USER__backgroundColor) !important;
}

:root[style*="--USER__backgroundColor"] *{
  background-color:transparent !important;
}

:root[style*="--USER__textColor"]{
  color:var(--USER__textColor) !important;
}

:root[style*="--USER__textColor"] *:not(a){
  color:inherit !important;
  background-color:transparent !important;
  border-color:currentcolor !important;
}

:root[style*="--USER__textColor"] svg text{
  fill:currentcolor !important;
  stroke:none !important;
}

:root[style*="--USER__linkColor"] a:link,
:root[style*="--USER__linkColor"] a:link *{
  color:var(--USER__linkColor) !important;
}

:root[style*="--USER__visitedColor"] a:visited,
:root[style*="--USER__visitedColor"] a:visited *{
  color:var(--USER__visitedColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::-moz-selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__lineLength"] body{
    max-height:var(--USER__lineLength) !important;
  }

:root[style*="--USER__fontFamily"]{
  font-family:var(--USER__fontFamily) !important;
}

:root[style*="--USER__fontFamily"] *{
  font-family:revert !important;
}

:root:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] body{
  zoom:var(--USER__fontSize) !important;
}

:root:not([style*="readium-deprecatedFontSize-on"])[style*="readium-iOSPatch-on"][style*="--USER__fontSize"] body{
  -webkit-text-size-adjust:var(--USER__fontSize) !important;
}

@supports selector(figure:has(> img)){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> img),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> video),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> svg),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> canvas),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> iframe),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> audio),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> img:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> video:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> svg:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> canvas:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> iframe:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> audio:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] table{
    zoom:calc(100% / var(--USER__fontSize)) !important;
  }

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figcaption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] caption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] td,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] th{
    zoom:var(--USER__fontSize) !important;
  }
}

@supports not (zoom: 1){

  :root[style*="--USER__fontSize"]{
    font-size:var(--USER__fontSize) !important;
  }
}

:root[style*="readium-deprecatedFontSize-on"][style*="--USER__fontSize"]{
  font-size:var(--USER__fontSize) !important;
}

:root[style*="--USER__lineHeight"]{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__lineHeight"] body,
:root[style*="--USER__lineHeight"] p,
:root[style*="--USER__lineHeight"] li,
:root[style*="--USER__lineHeight"] div{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__paraSpacing"] p{
  margin-right:var(--USER__paraSpacing) !important;
  margin-left:var(--USER__paraSpacing) !important;
}

:root[style*="--USER__fontWeight"] body{
  font-weight:var(--USER__fontWeight) !important;
}

:root[style*="--USER__fontWeight"] b,
:root[style*="--USER__fontWeight"] strong{
  font-weight:bolder;
}

:root[style*="--USER__fontWidth"] body{
  font-stretch:var(--USER__fontWidth) !important;
}

:root[style*="--USER__fontOpticalSizing"] body{
  font-optical-sizing:var(--USER__fontOpticalSizing) !important;
}

:root[style*="--USER__letterSpacing"] h1,
:root[style*="--USER__letterSpacing"] h2,
:root[style*="--USER__letterSpacing"] h3,
:root[style*="--USER__letterSpacing"] h4,
:root[style*="--USER__letterSpacing"] h5,
:root[style*="--USER__letterSpacing"] h6,
:root[style*="--USER__letterSpacing"] p,
:root[style*="--USER__letterSpacing"] li,
:root[style*="--USER__letterSpacing"] div,
:root[style*="--USER__letterSpacing"] dt,
:root[style*="--USER__letterSpacing"] dd{
  letter-spacing:var(--USER__letterSpacing) !important;
  font-variant:none !important;
}

:root[style*="readium-noRuby-on"] body rt,
:root[style*="readium-noRuby-on"] body rp{
  display:none;
}

:root[style*="readium-blend-on"] svg,
:root[style*="readium-blend-on"] img{
  background-color:transparent !important;
  mix-blend-mode:multiply !important;
}

:root[style*="--USER__darkenImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) !important;
  filter:brightness(var(--USER__darkenImages)) !important;
}

:root[style*="readium-darken-on"] img{
  -webkit-filter:brightness(80%) !important;
  filter:brightness(80%) !important;
}

:root[style*="--USER__invertImages"] img{
  -webkit-filter:invert(var(--USER__invertImages)) !important;
  filter:invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-invert-on"] img{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="--USER__darkenImages"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
  filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-darken-on"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(80%) invert(var(--USER__invertImages)) !important;
  filter:brightness(80%) invert(var(--USER__invertImages)) !important;
}

:root[style*="--USER__darkenImages"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
  filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
}

:root[style*="readium-darken-on"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(80%) invert(100%) !important;
  filter:brightness(80%) invert(100%) !important;
}

:root[style*="--USER__invertGaiji"] img[class*="gaiji"]{
  -webkit-filter:invert(var(--USER__invertGaiji)) !important;
  filter:invert(var(--USER__invertGaiji)) !important;
}

:root[style*="readium-invertGaiji-on"] img[class*="gaiji"]{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="readium-normalize-on"]{
  --USER__typeScale:1.2;
}

:root[style*="readium-normalize-on"] p,
:root[style*="readium-normalize-on"] li,
:root[style*="readium-normalize-on"] div,
:root[style*="readium-normalize-on"] pre,
:root[style*="readium-normalize-on"] dd{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] h1{
  font-size:1.75rem !important;
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h2{
  font-size:1.5rem !important;
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h3{
  font-size:1.25rem !important;
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h4,
:root[style*="readium-normalize-on"] h5,
:root[style*="readium-normalize-on"] h6{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] small{
  font-size:smaller !important;
}

:root[style*="readium-normalize-on"] sub,
:root[style*="readium-normalize-on"] sup{
  font-size:67.5% !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h1{
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h2{
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h3{
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-iPadOSPatch-on"] body{
  -webkit-text-size-adjust:none;
}

:root[style*="readium-iPadOSPatch-on"] p, 
:root[style*="readium-iPadOSPatch-on"] h1, 
:root[style*="readium-iPadOSPatch-on"] h2, 
:root[style*="readium-iPadOSPatch-on"] h3, 
:root[style*="readium-iPadOSPatch-on"] h4, 
:root[style*="readium-iPadOSPatch-on"] h5, 
:root[style*="readium-iPadOSPatch-on"] h6, 
:root[style*="readium-iPadOSPatch-on"] li, 
:root[style*="readium-iPadOSPatch-on"] th, 
:root[style*="readium-iPadOSPatch-on"] td, 
:root[style*="readium-iPadOSPatch-on"] dt, 
:root[style*="readium-iPadOSPatch-on"] dd, 
:root[style*="readium-iPadOSPatch-on"] pre, 
:root[style*="readium-iPadOSPatch-on"] address, 
:root[style*="readium-iPadOSPatch-on"] details, 
:root[style*="readium-iPadOSPatch-on"] summary,
:root[style*="readium-iPadOSPatch-on"] figcaption,
:root[style*="readium-iPadOSPatch-on"] div:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)),
:root[style*="readium-iPadOSPatch-on"] aside:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)){
  -webkit-text-zoom:reset;
}

:root[style*="readium-iPadOSPatch-on"] abbr, 
:root[style*="readium-iPadOSPatch-on"] b, 
:root[style*="readium-iPadOSPatch-on"] bdi, 
:root[style*="readium-iPadOSPatch-on"] bdo, 
:root[style*="readium-iPadOSPatch-on"] cite, 
:root[style*="readium-iPadOSPatch-on"] code, 
:root[style*="readium-iPadOSPatch-on"] dfn, 
:root[style*="readium-iPadOSPatch-on"] em, 
:root[style*="readium-iPadOSPatch-on"] i, 
:root[style*="readium-iPadOSPatch-on"] kbd, 
:root[style*="readium-iPadOSPatch-on"] mark, 
:root[style*="readium-iPadOSPatch-on"] q, 
:root[style*="readium-iPadOSPatch-on"] rp, 
:root[style*="readium-iPadOSPatch-on"] rt, 
:root[style*="readium-iPadOSPatch-on"] ruby, 
:root[style*="readium-iPadOSPatch-on"] s, 
:root[style*="readium-iPadOSPatch-on"] samp, 
:root[style*="readium-iPadOSPatch-on"] small, 
:root[style*="readium-iPadOSPatch-on"] span, 
:root[style*="readium-iPadOSPatch-on"] strong, 
:root[style*="readium-iPadOSPatch-on"] sub, 
:root[style*="readium-iPadOSPatch-on"] sup, 
:root[style*="readium-iPadOSPatch-on"] time, 
:root[style*="readium-iPadOSPatch-on"] u, 
:root[style*="readium-iPadOSPatch-on"] var{
  -webkit-text-zoom:normal;
}

:root[style*="readium-iPadOSPatch-on"] p:not(:has(b, cite, em, i, q, s, small, span, strong)):first-line{
  -webkit-text-zoom:normal;
}`});var rs={};nt(rs,{default:()=>qa});var qa,ss=ot(()=>{qa=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

@-ms-viewport{
  width:device-width;
}

@viewport{
  width:device-width;
  zoom:1;
}

:root{

  --RS__sans-serif-ja-v:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDGothic', 'Yu Gothic', '\uFF2D\uFF33\u30B4\u30B7\u30C3\u30AF', 'MS Gothic', sans-serif;

  --RS__serif-ja-v:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDMincho', 'Yu Mincho', '\uFF2D\uFF33\u660E\u671D', 'MS Mincho', serif;

  --RS__sans-serif-ja:'Hiragino Sans', 'Hiragino Kaku Gothic ProN', 'Hiragino Kaku Gothic Pro', '\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3', 'YuGothic', 'Yu Gothic Medium', 'BIZ UDPGothic', 'Yu Gothic', '\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF', 'MS PGothic', sans-serif;

  --RS__serif-ja:'Hiragino Mincho ProN', 'Hiragino Mincho Pro', 'YuMincho', 'BIZ UDPMincho', 'Yu Mincho', '\uFF2D\uFF33 \uFF30\u660E\u671D', 'MS PMincho', serif;

  --RS__monospaceTf:ui-monospace, 'Andale Mono', 'Cascadia Code', 'Source Code Pro', Menlo, Consolas, 'DejaVu Sans Mono', monospace;

  --RS__humanistTf:Seravek, Calibri, 'Gill Sans Nova', Roboto, Ubuntu, 'DejaVu Sans', source-sans-pro, sans-serif;

  --RS__sansTf:-ui-sans-serif, -apple-system, system-ui, BlinkMacSystemFont, 'Segoe UI Variable', 'Segoe UI', Inter, Roboto, 'Helvetica Neue', 'Arial Nova', 'Liberation Sans', Arial, sans-serif;

  --RS__modernTf:Athelas, Constantia, Charter, 'Bitstream Charter', Cambria, 'Georgia Pro', Georgia, serif;

  --RS__oldStyleTf:'Iowan Old Style', Sitka, 'Sitka Text', Palatino, 'Book Antiqua', 'URW Palladio L', P052, serif;

  --RS__zh-HK-lineHeightCompensation:1.167;

  --RS__zh-HK-baseFontFamily:'\u65B9\u9AD4', 'PingFang HK', '\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-TW-lineHeightCompensation:1.167;

  --RS__zh-TW-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-Hant-lineHeightCompensation:1.167;

  --RS__zh-Hant-baseFontFamily:'\u65B9\u9AD4', 'PingFang TC', '\u9ED1\u9AD4', 'Heiti TC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK TC', sans-serif;

  --RS__zh-lineHeightCompensation:1.167;

  --RS__zh-baseFontFamily:'\u65B9\u4F53', 'PingFang SC', '\u9ED1\u4F53', 'Heiti SC', 'Microsoft JhengHei UI', 'Microsoft JhengHei', Roboto, Noto, 'Noto Sans CJK SC', sans-serif;

  --RS__th-lineHeightCompensation:1.067;

  --RS__th-baseFontFamily:Thonburi, 'Leelawadee UI', 'Cordia New', Roboto, Noto, 'Noto Sans Thai', sans-serif;

  --RS__te-baseFontFamily:'Kohinoor Telugu', 'Telugu Sangam MN', 'Nirmala UI', Gautami, Roboto, Noto, 'Noto Sans Telugu', sans-serif;

  --RS__ta-lineHeightCompensation:1.067;

  --RS__ta-baseFontFamily:'Tamil Sangam MN', 'Nirmala UI', Latha, Roboto, Noto, 'Noto Sans Tamil', sans-serif;

  --RS__si-lineHeightCompensation:1.167;

  --RS__si-baseFontFamily:'Sinhala Sangam MN', 'Nirmala UI', 'Iskoola Pota', Roboto, Noto, 'Noto Sans Sinhala', sans-serif;

  --RS__pa-lineHeightCompensation:1.1;

  --RS__pa-baseFontFamily:'Gurmukhi MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Gurmukhi', sans-serif;

  --RS__or-lineHeightCompensation:1.167;

  --RS__or-baseFontFamily:'Oriya Sangam MN', 'Nirmala UI', Kalinga, Roboto, Noto, 'Noto Sans Oriya', sans-serif;

  --RS__ml-lineHeightCompensation:1.067;

  --RS__ml-baseFontFamily:'Malayalam Sangam MN', 'Nirmala UI', Kartika, Roboto, Noto, 'Noto Sans Malayalam', sans-serif;

  --RS__lo-baseFontFamily:'Lao Sangam MN', 'Leelawadee UI', 'Lao UI', Roboto, Noto, 'Noto Sans Lao', sans-serif;

  --RS__ko-lineHeightCompensation:1.167;

  --RS__ko-baseFontFamily:'Nanum Gothic', 'Apple SD Gothic Neo', 'Malgun Gothic', Roboto, Noto, 'Noto Sans CJK KR', sans-serif;

  --RS__kn-lineHeightCompensation:1.1;

  --RS__kn-baseFontFamily:'Kannada Sangam MN', 'Nirmala UI', Tunga, Roboto, Noto, 'Noto Sans Kannada', sans-serif;

  --RS__km-lineHeightCompensation:1.067;

  --RS__km-baseFontFamily:'Khmer Sangam MN', 'Leelawadee UI', 'Khmer UI', Roboto, Noto, 'Noto Sans Khmer', sans-serif;

  --RS__ja-lineHeightCompensation:1.167;

  --RS__ja-baseFontFamily:YuGothic, 'Hiragino Maru Gothic ProN', 'Hiragino Sans', 'Yu Gothic UI', 'Meiryo UI', 'MS Gothic', Roboto, Noto, 'Noto Sans CJK JP', sans-serif;

  --RS__iu-baseFontFamily:'Euphemia UCAS', Euphemia, Roboto, Noto, 'Noto Sans Canadian Aboriginal', sans-serif;

  --RS__hy-baseFontFamily:Mshtakan, Sylfaen, Roboto, Noto, 'Noto Serif Armenian', serif;

  --RS__hi-lineHeightCompensation:1.1;

  --RS__hi-baseFontFamily:'Kohinoor Devanagari', 'Devanagari Sangam MN', Kokila, 'Nirmala UI', Roboto, Noto, 'Noto Sans Devanagari', sans-serif;

  --RS__he-lineHeightCompensation:1.1;

  --RS__he-baseFontFamily:'New Peninim MT', 'Arial Hebrew', Gisha, 'Times New Roman', Roboto, Noto, 'Noto Sans Hebrew', sans-serif;

  --RS__gu-lineHeightCompensation:1.167;

  --RS__gu-baseFontFamily:'Gujarati Sangam MN', 'Nirmala UI', Shruti, Roboto, Noto, 'Noto Sans Gujarati', sans-serif;

  --RS__fa-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__chr-lineHeightCompensation:1.167;

  --RS__chr-baseFontFamily:'Plantagenet Cherokee', Roboto, Noto, 'Noto Sans Cherokee';

  --RS__bo-baseFontFamily:Kailasa, 'Microsoft Himalaya', Roboto, Noto, 'Noto Sans Tibetan', sans-serif;

  --RS__bn-lineHeightCompensation:1.067;

  --RS__bn-baseFontFamily:'Kohinoor Bangla', 'Bangla Sangam MN', Vrinda, Roboto, Noto, 'Noto Sans Bengali', sans-serif;

  --RS__ar-baseFontFamily:'Geeza Pro', 'Arabic Typesetting', Roboto, Noto, 'Noto Naskh Arabic', 'Times New Roman', serif;

  --RS__am-lineHeightCompensation:1.167;

  --RS__am-baseFontFamily:Kefa, Nyala, Roboto, Noto, 'Noto Sans Ethiopic', serif;

  --RS__latin-lineHeightCompensation:1;

  --RS__latin-baseFontFamily:var(--RS__oldStyleTf);
  --RS__baseFontFamily:var(--RS__latin-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__latin-lineHeightCompensation);
  --RS__baseLineHeight:calc(1.5 * var(--RS__lineHeightCompensation));

  --RS__selectionTextColor:inherit;

  --RS__selectionBackgroundColor:#b4d8fe;

  --RS__visitedColor:#551A8B;

  --RS__linkColor:#0000EE;

  --RS__textColor:#121212;

  --RS__backgroundColor:#FFFFFF;
  color:var(--RS__textColor) !important;

  background-color:var(--RS__backgroundColor) !important;
}

::-moz-selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

::selection{
  color:var(--RS__selectionTextColor);
  background-color:var(--RS__selectionBackgroundColor);
}

html{
  font-family:var(--RS__baseFontFamily);
  line-height:1.6;
  line-height:var(--RS__baseLineHeight);
  text-rendering:optimizelegibility;
}

h1, h2, h3{
  line-height:normal;
}

:lang(ja),
:lang(zh),
:lang(ko){
  word-wrap:break-word;
  -webkit-line-break:strict;
  -epub-line-break:strict;
  line-break:strict;
}

math{
  font-family:"Latin Modern Math", "STIX Two Math", "XITS Math", "STIX Math", "Libertinus Math", "TeX Gyre Termes Math", "TeX Gyre Bonum Math", "TeX Gyre Schola", "DejaVu Math TeX Gyre", "TeX Gyre Pagella Math", "Asana Math", "Cambria Math", "Lucida Bright Math", "Minion Math", STIXGeneral, STIXSizeOneSym, Symbol, "Times New Roman", serif;
}

:lang(am){
  --RS__baseFontFamily:var(--RS__am-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__am-lineHeightCompensation);
}

:lang(ar){
  --RS__baseFontFamily:var(--RS__ar-baseFontFamily);
}

:lang(bn){
  --RS__baseFontFamily:var(--RS__bn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__bn-lineHeightCompensation);
}

:lang(bo){
  --RS__baseFontFamily:var(--RS__bo-baseFontFamily);
}

:lang(chr){
  --RS__baseFontFamily:var(--RS__chr-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__chr-lineHeightCompensation);
}

:lang(fa){
  --RS__baseFontFamily:var(--RS__fa-baseFontFamily);
}

:lang(gu){
  --RS__baseFontFamily:var(--RS__gu-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__gu-lineHeightCompensation);
}

:lang(he){
  --RS__baseFontFamily:var(--RS__he-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__he-lineHeightCompensation);
}

:lang(hi){
  --RS__baseFontFamily:var(--RS__hi-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__hi-lineHeightCompensation);
}

:lang(hy){
  --RS__baseFontFamily:var(--RS__hy-baseFontFamily);
}

:lang(iu){
  --RS__baseFontFamily:var(--RS__iu-baseFontFamily);
}

:lang(ja){
  --RS__baseFontFamily:var(--RS__ja-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ja-lineHeightCompensation);
}

:lang(km){
  --RS__baseFontFamily:var(--RS__km-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__km-lineHeightCompensation);
}

:lang(kn){
  --RS__baseFontFamily:var(--RS__kn-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__kn-lineHeightCompensation);
}

:lang(ko){
  --RS__baseFontFamily:var(--RS__ko-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ko-lineHeightCompensation);
}

:lang(lo){
  --RS__baseFontFamily:var(--RS__lo-baseFontFamily);
}

:lang(ml){
  --RS__baseFontFamily:var(--RS__ml-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ml-lineHeightCompensation);
}

:lang(or){
  --RS__baseFontFamily:var(--RS__or-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__or-lineHeightCompensation);
}

:lang(pa){
  --RS__baseFontFamily:var(--RS__pa-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__pa-lineHeightCompensation);
}

:lang(si){
  --RS__baseFontFamily:var(--RS__si-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__si-lineHeightCompensation);
}

:lang(ta){
  --RS__baseFontFamily:var(--RS__ta-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__ta-lineHeightCompensation);
}

:lang(te){
  --RS__baseFontFamily:var(--RS__te-baseFontFamily);
}

:lang(th){
  --RS__baseFontFamily:var(--RS__th-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__th-lineHeightCompensation);
}

:lang(zh){
  --RS__baseFontFamily:var(--RS__zh-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-lineHeightCompensation);
}

:lang(zh-Hant){
  --RS__baseFontFamily:var(--RS__zh-Hant-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-Hant-lineHeightCompensation);
}

:lang(zh-TW){
  --RS__baseFontFamily:var(--RS__zh-TW-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-TW-lineHeightCompensation);
}

:lang(zh-HK){
  --RS__baseFontFamily:var(--RS__zh-HK-baseFontFamily);
  --RS__lineHeightCompensation:var(--RS__zh-HK-lineHeightCompensation);
}

body{
  widows:2;
  orphans:2;
}

figcaption, th, td{
  widows:1;
  orphans:1;
}

h2, h3, h4, h5, h6, dt,
hr, caption{
  -webkit-column-break-after:avoid;
  page-break-after:avoid;
  break-after:avoid;
}

h1, h2, h3, h4, h5, h6, dt,
figure, tr{
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

body{
  -webkit-hyphenate-character:"\\002D";
  -moz-hyphenate-character:"\\002D";
  -ms-hyphenate-character:"\\002D";
  hyphenate-character:"\\002D";
  -webkit-hyphenate-limit-lines:3;
  -ms-hyphenate-limit-lines:3;
  hyphenate-limit-lines:3;
}

h1, h2, h3, h4, h5, h6, dt,
figcaption, pre, caption, address,
center, code, var{
  -ms-hyphens:none;
  -moz-hyphens:none;
  -webkit-hyphens:none;
  -epub-hyphens:none;
  hyphens:none;
}

body{
  font-variant-numeric:oldstyle-nums proportional-nums;
}

:lang(ja) body,
:lang(zh) body,
:lang(ko) body{
  font-variant-numeric:lining-nums proportional-nums;
}

h1, h2, h3, h4, h5, h6, dt{
  font-variant-numeric:lining-nums proportional-nums;
}

table{
  font-variant-numeric:lining-nums tabular-nums;
}

code, var{
  font-variant-ligatures:none;
  font-variant-numeric:lining-nums tabular-nums slashed-zero;
}

rt{
  font-variant-east-asian:ruby;
}

:lang(ar){
  font-variant-ligatures:common-ligatures;
}

:lang(ko){
  font-kerning:normal;
}

hr{
  color:inherit;
  border-color:currentcolor;
}

table, th, td{
  border-color:currentcolor;
}

figure, blockquote{
  margin:1em 5%;
}

ul, ol{
  padding-left:5%;
}

dd{
  margin-left:5%;
}

pre{
  white-space:pre-wrap;
  -ms-tab-size:2;
  -moz-tab-size:2;
  -webkit-tab-size:2;
  tab-size:2;
}

abbr[title], acronym[title]{
  text-decoration:dotted underline;
}

nobr wbr{
  white-space:normal;
}

ruby > rt, ruby > rp{
  -webkit-user-select:none;
  -moz-user-select:none;
  -ms-user-select:none;
  user-select:none;
}

*:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)),
*:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)),
*:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)),
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) cite, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) dfn, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) em, 
:lang(ja):not(:lang(ja-Latn)):not(:lang(ja-Cyrl)) i,
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) cite, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) dfn, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) em, 
:lang(zh):not(:lang(zh-Latn)):not(:lang(zh-Cyrl)) i,
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) cite, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) dfn, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) em, 
:lang(ko):not(:lang(ko-Latn)):not(:lang(ko-Cyrl)) i{
  font-style:normal;
}

:lang(ja) a,
:lang(zh) a,
:lang(ko) a{
  text-decoration:none;
}

:root{
  --RS__maxMediaWidth:100%;
  --RS__maxMediaHeight:95vh;
  --RS__boxSizingMedia:border-box;
  --RS__boxSizingTable:border-box;
}

a, a span, span a, h1, h2, h3, h4, h5, h6{
  word-wrap:break-word;
}

div{
  max-width:var(--RS__maxMediaWidth);
}

img, svg|svg, video{
  object-fit:contain;

  width:auto;
  height:auto;
  max-width:var(--RS__maxMediaWidth);
  max-height:var(--RS__maxMediaHeight) !important;
  box-sizing:var(--RS__boxSizingMedia);
  -webkit-column-break-inside:avoid;
  page-break-inside:avoid;
  break-inside:avoid;
}

@supports (zoom: 1) and (not ((-webkit-column-axis: horizontal) and (-webkit-column-progression: normal))){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] img,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] svg|svg,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] video{
    zoom:calc(100% / var(--USER__fontSize));
  }
}

audio{
    max-width:100%;
    -webkit-column-break-inside:avoid;
    page-break-inside:avoid;
    break-inside:avoid;
  }

table{
  max-width:var(--RS__maxMediaWidth);
  box-sizing:var(--RS__boxSizingTable);
}`});var as={};nt(as,{default:()=>Ja});var Ja,ls=ot(()=>{Ja=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{
  --RS__compFontFamily:var(--RS__baseFontFamily);
  --RS__codeFontFamily:var(--RS__monospaceTf);

  --RS__typeScale:1.125;
  --RS__baseFontSize:100%;

  --RS__flowSpacing:1.5rem;
  --RS__paraSpacing:0;
  --RS__paraIndent:1em;

  --RS__linkColor:#0000EE;
  --RS__visitedColor:#551A8B;

  --RS__primaryColor:;
  --RS__secondaryColor:;
}

body{
  font-size:var(--RS__baseFontSize);
}

h1, h2, h3, h4, h5, h6{
  font-family:var(--RS__compFontFamily);
}

blockquote, figure, p, pre,
aside, footer, form, hr{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

p{
  margin-top:var(--RS__paraSpacing);
  margin-bottom:var(--RS__paraSpacing);
  text-indent:var(--RS__paraIndent);
}

h1 + p, h2 + p, h3 + p, h4 + p, h5 + p, h6 + p,
hr + p{
  text-indent:0;
}

pre{
  font-family:var(--RS__codeFontFamily);
}

code, kbd, samp, tt{
  font-family:var(--RS__codeFontFamily);
}

sub, sup{
  position:relative;
  font-size:67.5%;
  line-height:1;
}

sub{
  bottom:-0.2ex;
}

sup{
  bottom:0;
}

:link{
  color:var(--RS__linkColor);
}

:visited{
  color:var(--RS__visitedColor);
}

h1{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:calc(var(--RS__flowSpacing) * 2);
  font-size:calc(((1em * var(--RS__typeScale)) * var(--RS__typeScale)) * var(--RS__typeScale));
}

h2{
  margin-top:calc(var(--RS__flowSpacing) * 2);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc((1em * var(--RS__typeScale)) * var(--RS__typeScale));
}

h3{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:calc(1em * var(--RS__typeScale));
}

h4{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:1em;
}

h5{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
  font-size:1em;
  font-variant:small-caps;
}

h6{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:0;
  font-size:1em;
  text-transform:lowercase;
  font-variant:small-caps;
}

dl, ol, ul{
  margin-top:var(--RS__flowSpacing);
  margin-bottom:var(--RS__flowSpacing);
}

table{
  margin:var(--RS__flowSpacing) 0;
  border:1px solid currentcolor;
  border-collapse:collapse;
  empty-cells:show;
}

thead, tbody, tfoot, table > tr{
  vertical-align:top;
}

th{
  text-align:left;
}

th, td{
  padding:4px;
  border:1px solid currentcolor;
}`});var hs={};nt(hs,{default:()=>Za});var Za,cs=ot(()=>{Za=`/*!
 * Readium CSS v.2.0.5
 * Copyright (c) 2017\u20132026. Readium Foundation. All rights reserved.
 * Use of this source code is governed by a BSD-style license which is detailed in the
 * LICENSE file present in the project repository where this source code is maintained.
 * Core maintainer: Jiminy Panoz <jiminy.panoz@edrlab.org> 
 * Contributors: 
 * Daniel Weck
 * Hadrien Gardeur
 * Innovimax
 * L. Le Meur
 * Mickae\u0308l Menu
 * k_taka
 */

@namespace url("http://www.w3.org/1999/xhtml");

@namespace epub url("http://www.idpf.org/2007/ops");

@namespace m url("http://www.w3.org/1998/Math/MathML");

@namespace svg url("http://www.w3.org/2000/svg");

:root{

  --RS__viewportWidth:100%;

  --RS__pageGutter:0;

  --RS__defaultLineLength:100%;

  --RS__colGap:0;

  --RS__colCount:1;

  --RS__colWidth:100vw;
}

@page{
  margin:0 !important;
}

:root{
  position:relative;

  -webkit-column-width:var(--RS__colWidth);
  -moz-column-width:var(--RS__colWidth);
  column-width:var(--RS__colWidth);
  -webkit-column-count:var(--RS__colCount);
  -moz-column-count:var(--RS__colCount);
  column-count:var(--RS__colCount);

  -webkit-column-gap:var(--RS__colGap);
  -moz-column-gap:var(--RS__colGap);
  column-gap:var(--RS__colGap);
  -moz-column-fill:auto;
  column-fill:auto;
  width:var(--RS__viewportWidth);
  height:100vh;
  max-width:var(--RS__viewportWidth);
  max-height:100vh;
  min-width:var(--RS__viewportWidth);
  min-height:100vh;
  padding:0 !important;
  margin:0 !important;
  font-size:1rem !important;
  box-sizing:border-box;
  -webkit-touch-callout:none;
}

body{
  width:100%;
  max-width:var(--RS__defaultLineLength) !important;
  margin:0 auto !important;
  box-sizing:border-box;
}

:root:not([style*="readium-scroll-on"]) body{
  padding:0 var(--RS__pageGutter) !important;
}

:root:not([style*="readium-noOverflow-on"]) body{
  overflow:hidden;
}

@supports (overflow: clip){

   :root:not([style*="readium-noOverflow-on"]){
      overflow:clip;
   }

   :root:not([style*="readium-noOverflow-on"]) body{
      overflow:clip;
      overflow-clip-margin:content-box;
   }
}

:root[style*="readium-scroll-on"]{
  -webkit-columns:auto auto !important;
  -moz-columns:auto auto !important;
  columns:auto auto !important;
  width:auto !important;
  height:auto !important;
  max-width:none !important;
  max-height:none !important;
  min-width:0 !important;
  min-height:0 !important;
}

:root[style*="readium-scroll-on"] body{
  max-width:var(--RS__defaultLineLength) !important;
  box-sizing:border-box !important;
}

:root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
  overflow:auto;
}

@supports (overflow: clip){

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]){
     overflow:auto;
  }

  :root[style*="readium-scroll-on"]:not([style*="readium-noOverflow-on"]) body{
     overflow:clip;
  }
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingTop"] body{
  padding-top:var(--RS__scrollPaddingTop) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingBottom"] body{
  padding-bottom:var(--RS__scrollPaddingBottom) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingLeft"] body{
  padding-left:var(--RS__scrollPaddingLeft) !important;
}

:root[style*="readium-scroll-on"][style*="--RS__scrollPaddingRight"] body{
  padding-right:var(--RS__scrollPaddingRight) !important;
}

:root[style*="--USER__backgroundColor"]{
  background-color:var(--USER__backgroundColor) !important;
}

:root[style*="--USER__backgroundColor"] *{
  background-color:transparent !important;
}

:root[style*="--USER__textColor"]{
  color:var(--USER__textColor) !important;
}

:root[style*="--USER__textColor"] *:not(a){
  color:inherit !important;
  background-color:transparent !important;
  border-color:currentcolor !important;
}

:root[style*="--USER__textColor"] svg text{
  fill:currentcolor !important;
  stroke:none !important;
}

:root[style*="--USER__linkColor"] a:link,
:root[style*="--USER__linkColor"] a:link *{
  color:var(--USER__linkColor) !important;
}

:root[style*="--USER__visitedColor"] a:visited,
:root[style*="--USER__visitedColor"] a:visited *{
  color:var(--USER__visitedColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::-moz-selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__selectionBackgroundColor"][style*="--USER__selectionTextColor"] ::selection{
  color:var(--USER__selectionTextColor) !important;
  background-color:var(--USER__selectionBackgroundColor) !important;
}

:root[style*="--USER__colCount"]{
  -webkit-column-count:var(--USER__colCount);
  -moz-column-count:var(--USER__colCount);
  column-count:var(--USER__colCount);

  --RS__colWidth:auto;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"]{
  -webkit-column-count:1;
  -moz-column-count:1;
  column-count:1;
}

:root[style*="--USER__colCount: 0"],
:root[style*="--USER__colCount:0"],
:root[style*="--USER__colCount: 1"],
:root[style*="--USER__colCount:1"]{
  --RS__colWidth:100vw;
}

:root[style*="--USER__lineLength"] body{
    max-width:var(--USER__lineLength) !important;
  }

:root[style*="--USER__textAlign"]{
  text-align:var(--USER__textAlign);
}

:root[style*="--USER__textAlign"] body,
:root[style*="--USER__textAlign"] p:not(
  blockquote p,
  figcaption p,
  header p,
  hgroup p,
  :root[style*="readium-experimentalHeaderFiltering-on"] p[class*="title"],
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > h1 + p,
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > p:has(+ h1)
),
:root[style*="--USER__textAlign"] li,
:root[style*="--USER__textAlign"] dd{
  text-align:var(--USER__textAlign) !important;
  -moz-text-align-last:auto !important;
  -epub-text-align-last:auto !important;
  text-align-last:auto !important;
}

:root[style*="--USER__bodyHyphens"]{
  -webkit-hyphens:var(--USER__bodyHyphens) !important;
  -moz-hyphens:var(--USER__bodyHyphens) !important;
  -ms-hyphens:var(--USER__bodyHyphens) !important;
  -epub-hyphens:var(--USER__bodyHyphens) !important;
  hyphens:var(--USER__bodyHyphens) !important;
}

:root[style*="--USER__bodyHyphens"] body,
:root[style*="--USER__bodyHyphens"] p,
:root[style*="--USER__bodyHyphens"] li,
:root[style*="--USER__bodyHyphens"] div,
:root[style*="--USER__bodyHyphens"] dd{
  -webkit-hyphens:var(--USER__bodyHyphens) !important;
  -moz-hyphens:var(--USER__bodyHyphens) !important;
  -ms-hyphens:var(--USER__bodyHyphens) !important;
  -epub-hyphens:var(--USER__bodyHyphens) !important;
  hyphens:var(--USER__bodyHyphens) !important;
}

:root[style*="--USER__fontFamily"]{
  font-family:var(--USER__fontFamily) !important;
}

:root[style*="--USER__fontFamily"] *{
  font-family:revert !important;
}

:root[style*="readium-a11y-on"]{
  font-style:normal !important;
  font-weight:normal !important;
}

:root[style*="readium-a11y-on"] body *:not(code):not(var):not(kbd):not(samp){
  font-family:inherit !important;
  font-style:inherit !important;
  font-weight:inherit !important;
}

:root[style*="readium-a11y-on"] body *:not(a){
  text-decoration:none !important;
}

:root[style*="readium-a11y-on"] body *{
  font-variant-caps:normal !important;
  font-variant-numeric:normal !important;
  font-variant-position:normal !important;
}

:root[style*="readium-a11y-on"] sup,
:root[style*="readium-a11y-on"] sub{
  font-size:1rem !important;
  vertical-align:baseline !important;
}

:root:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] body{
  zoom:var(--USER__fontSize) !important;
}

:root:not([style*="readium-deprecatedFontSize-on"])[style*="readium-iOSPatch-on"][style*="--USER__fontSize"] body{
  -webkit-text-size-adjust:var(--USER__fontSize) !important;
}

@supports selector(figure:has(> img)){

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> img),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> video),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> svg),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> canvas),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> iframe),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figure:has(> audio),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> img:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> video:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> svg:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> canvas:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> iframe:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] div:has(> audio:only-child),
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] table{
    zoom:calc(100% / var(--USER__fontSize)) !important;
  }

  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] figcaption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] caption,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] td,
  :root[style*="readium-experimentalZoom-on"]:not([style*="readium-deprecatedFontSize-on"]):not([style*="readium-iOSPatch-on"])[style*="--USER__fontSize"] th{
    zoom:var(--USER__fontSize) !important;
  }
}

@supports not (zoom: 1){

  :root[style*="--USER__fontSize"]{
    font-size:var(--USER__fontSize) !important;
  }
}

:root[style*="readium-deprecatedFontSize-on"][style*="--USER__fontSize"]{
  font-size:var(--USER__fontSize) !important;
}

:root[style*="--USER__lineHeight"]{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__lineHeight"] body,
:root[style*="--USER__lineHeight"] p,
:root[style*="--USER__lineHeight"] li,
:root[style*="--USER__lineHeight"] div{
  line-height:var(--USER__lineHeight) !important;
}

:root[style*="--USER__paraSpacing"] p{
  margin-top:var(--USER__paraSpacing) !important;
  margin-bottom:var(--USER__paraSpacing) !important;
}

:root[style*="--USER__paraIndent"] p:not(
  blockquote p,
  figcaption p,
  header p,
  hgroup p,
  :root[style*="readium-experimentalHeaderFiltering-on"] p[class*="title"],
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > h1 + p,
  :root[style*="readium-experimentalHeaderFiltering-on"] div:has(+ *) > p:has(+ h1)
){
  text-indent:var(--USER__paraIndent) !important;
}

:root[style*="--USER__paraIndent"] p *{
  text-indent:0 !important;
}

:root[style*="--USER__wordSpacing"] h1,
:root[style*="--USER__wordSpacing"] h2,
:root[style*="--USER__wordSpacing"] h3,
:root[style*="--USER__wordSpacing"] h4,
:root[style*="--USER__wordSpacing"] h5,
:root[style*="--USER__wordSpacing"] h6,
:root[style*="--USER__wordSpacing"] p,
:root[style*="--USER__wordSpacing"] li,
:root[style*="--USER__wordSpacing"] div,
:root[style*="--USER__wordSpacing"] dt,
:root[style*="--USER__wordSpacing"] dd{
  word-spacing:var(--USER__wordSpacing) !important;
}

:root[style*="--USER__letterSpacing"] h1,
:root[style*="--USER__letterSpacing"] h2,
:root[style*="--USER__letterSpacing"] h3,
:root[style*="--USER__letterSpacing"] h4,
:root[style*="--USER__letterSpacing"] h5,
:root[style*="--USER__letterSpacing"] h6,
:root[style*="--USER__letterSpacing"] p,
:root[style*="--USER__letterSpacing"] li,
:root[style*="--USER__letterSpacing"] div,
:root[style*="--USER__letterSpacing"] dt,
:root[style*="--USER__letterSpacing"] dd{
  letter-spacing:var(--USER__letterSpacing) !important;
  font-variant:none !important;
}

:root[style*="--USER__ligatures"]{
  font-variant-ligatures:var(--USER__ligatures) !important;
}

:root[style*="--USER__ligatures"] *{
  font-variant-ligatures:inherit !important;
}

:root[style*="--USER__fontWeight"] body{
  font-weight:var(--USER__fontWeight) !important;
}

:root[style*="--USER__fontWeight"] b,
:root[style*="--USER__fontWeight"] strong{
  font-weight:bolder;
}

:root[style*="--USER__fontWidth"] body{
  font-stretch:var(--USER__fontWidth) !important;
}

:root[style*="--USER__fontOpticalSizing"] body{
  font-optical-sizing:var(--USER__fontOpticalSizing) !important;
}

:root[style*="readium-blend-on"] svg,
:root[style*="readium-blend-on"] img{
  background-color:transparent !important;
  mix-blend-mode:multiply !important;
}

:root[style*="--USER__darkenImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) !important;
  filter:brightness(var(--USER__darkenImages)) !important;
}

:root[style*="readium-darken-on"] img{
  -webkit-filter:brightness(80%) !important;
  filter:brightness(80%) !important;
}

:root[style*="--USER__invertImages"] img{
  -webkit-filter:invert(var(--USER__invertImages)) !important;
  filter:invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-invert-on"] img{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="--USER__darkenImages"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
  filter:brightness(var(--USER__darkenImages)) invert(var(--USER__invertImages)) !important;
}

:root[style*="readium-darken-on"][style*="--USER__invertImages"] img{
  -webkit-filter:brightness(80%) invert(var(--USER__invertImages)) !important;
  filter:brightness(80%) invert(var(--USER__invertImages)) !important;
}

:root[style*="--USER__darkenImages"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
  filter:brightness(var(--USER__darkenImages)) invert(100%) !important;
}

:root[style*="readium-darken-on"][style*="readium-invert-on"] img{
  -webkit-filter:brightness(80%) invert(100%) !important;
  filter:brightness(80%) invert(100%) !important;
}

:root[style*="--USER__invertGaiji"] img[class*="gaiji"]{
  -webkit-filter:invert(var(--USER__invertGaiji)) !important;
  filter:invert(var(--USER__invertGaiji)) !important;
}

:root[style*="readium-invertGaiji-on"] img[class*="gaiji"]{
  -webkit-filter:invert(100%) !important;
  filter:invert(100%) !important;
}

:root[style*="readium-normalize-on"]{
  --USER__typeScale:1.2;
}

:root[style*="readium-normalize-on"] p,
:root[style*="readium-normalize-on"] li,
:root[style*="readium-normalize-on"] div,
:root[style*="readium-normalize-on"] pre,
:root[style*="readium-normalize-on"] dd{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] h1{
  font-size:1.75rem !important;
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h2{
  font-size:1.5rem !important;
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h3{
  font-size:1.25rem !important;
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"] h4,
:root[style*="readium-normalize-on"] h5,
:root[style*="readium-normalize-on"] h6{
  font-size:1rem !important;
}

:root[style*="readium-normalize-on"] small{
  font-size:smaller !important;
}

:root[style*="readium-normalize-on"] sub,
:root[style*="readium-normalize-on"] sup{
  font-size:67.5% !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h1{
  font-size:calc(((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h2{
  font-size:calc((1rem * var(--USER__typeScale)) * var(--USER__typeScale)) !important;
}

:root[style*="readium-normalize-on"][style*="--USER__typeScale"] h3{
  font-size:calc(1rem * var(--USER__typeScale)) !important;
}

:root[style*="readium-iPadOSPatch-on"] body{
  -webkit-text-size-adjust:none;
}

:root[style*="readium-iPadOSPatch-on"] p, 
:root[style*="readium-iPadOSPatch-on"] h1, 
:root[style*="readium-iPadOSPatch-on"] h2, 
:root[style*="readium-iPadOSPatch-on"] h3, 
:root[style*="readium-iPadOSPatch-on"] h4, 
:root[style*="readium-iPadOSPatch-on"] h5, 
:root[style*="readium-iPadOSPatch-on"] h6, 
:root[style*="readium-iPadOSPatch-on"] li, 
:root[style*="readium-iPadOSPatch-on"] th, 
:root[style*="readium-iPadOSPatch-on"] td, 
:root[style*="readium-iPadOSPatch-on"] dt, 
:root[style*="readium-iPadOSPatch-on"] dd, 
:root[style*="readium-iPadOSPatch-on"] pre, 
:root[style*="readium-iPadOSPatch-on"] address, 
:root[style*="readium-iPadOSPatch-on"] details, 
:root[style*="readium-iPadOSPatch-on"] summary,
:root[style*="readium-iPadOSPatch-on"] figcaption,
:root[style*="readium-iPadOSPatch-on"] div:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)),
:root[style*="readium-iPadOSPatch-on"] aside:not(:has(p, h1, h2, h3, h4, h5, h6, li, th, td, dt, dd, pre, address, aside, details, figcaption, summary)){
  -webkit-text-zoom:reset;
}

:root[style*="readium-iPadOSPatch-on"] abbr, 
:root[style*="readium-iPadOSPatch-on"] b, 
:root[style*="readium-iPadOSPatch-on"] bdi, 
:root[style*="readium-iPadOSPatch-on"] bdo, 
:root[style*="readium-iPadOSPatch-on"] cite, 
:root[style*="readium-iPadOSPatch-on"] code, 
:root[style*="readium-iPadOSPatch-on"] dfn, 
:root[style*="readium-iPadOSPatch-on"] em, 
:root[style*="readium-iPadOSPatch-on"] i, 
:root[style*="readium-iPadOSPatch-on"] kbd, 
:root[style*="readium-iPadOSPatch-on"] mark, 
:root[style*="readium-iPadOSPatch-on"] q, 
:root[style*="readium-iPadOSPatch-on"] rp, 
:root[style*="readium-iPadOSPatch-on"] rt, 
:root[style*="readium-iPadOSPatch-on"] ruby, 
:root[style*="readium-iPadOSPatch-on"] s, 
:root[style*="readium-iPadOSPatch-on"] samp, 
:root[style*="readium-iPadOSPatch-on"] small, 
:root[style*="readium-iPadOSPatch-on"] span, 
:root[style*="readium-iPadOSPatch-on"] strong, 
:root[style*="readium-iPadOSPatch-on"] sub, 
:root[style*="readium-iPadOSPatch-on"] sup, 
:root[style*="readium-iPadOSPatch-on"] time, 
:root[style*="readium-iPadOSPatch-on"] u, 
:root[style*="readium-iPadOSPatch-on"] var{
  -webkit-text-zoom:normal;
}

:root[style*="readium-iPadOSPatch-on"] p:not(:has(b, cite, em, i, q, s, small, span, strong)):first-line{
  -webkit-text-zoom:normal;
}`});var ds={};nt(ds,{default:()=>Qa});var Qa,ms=ot(()=>{Qa=`/* Readium CSS
   EBPAJ Fonts Patch module

   A stylesheet improving EBPAJ @font-face declarations to cover all platforms

   Repo: https://github.com/readium/css */

/* EBPAJ template only references fonts from MS Windows\u2026
   so we must reference fonts from other platforms
   and override authors\u2019 stylesheets.
   What we do there is keeping their default value and providing fallbacks.

   /!\\ /!\\ /!\\ /!\\ /!\\
   FYI, you might want to load this polyfill only if you find
   one of the following metadata items in the OPF package:
   - version 1:
     <dc:description id="ebpaj-guide">ebpaj-guide-1.0</dc:description>
   - version 1.1:
     <meta property="ebpaj:guide-version">1.1</meta>
*/

/* 
   Hiragino PostScript Font name lists:
   https://www.screen.co.jp/ga_product/sento/support/QA/ss_psname.html
*/

/* \u6A2A\u7D44\u307F\u7528 (horizontal writing) */

@font-face {
  font-family: "serif-ja";
  src: local("\uFF2D\uFF33 \uFF30\u660E\u671D"), /* for IE */
      local("MS PMincho"), /* \uFF2D\uFF33 \uFF30\u660E\u671D */
      local("HiraMinProN-W3"), local("Hiragino Mincho ProN"), /* \u30D2\u30E9\u30AE\u30CE\u660E\u671D ProN W3 */
      local("HiraMinPro-W3"), local("Hiragino Mincho Pro"), /* \u30D2\u30E9\u30AE\u30CE\u660E\u671D Pro W3 */
      local("YuMin-Medium"), local("YuMincho"), /* \u6E38\u660E\u671D\u4F53(macOS) */
      local("Yu Mincho"), /* \u6E38\u660E\u671D(Windows) */
      local("BIZ UDPMincho"); /* BIZ UDP\u660E\u671D */
}

@font-face {
  font-family: "sans-serif-ja";
  src: local("\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF"), /* for IE */
       local("MS PGothic"), /* \uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF */
       local("HiraginoSans-W3"), local("Hiragino Sans"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4\u30B7\u30C3\u30AF */
       local("HiraKakuProN-W3"), local("Hiragino Kaku Gothic ProN"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 ProN W3 */
       local("HiraKakuPro-W3"), local("Hiragino Kaku Gothic Pro"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 Pro W3 */
       local("\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3"), /* for old  Safari */
       local("HiraginoKaku-W3-90msp-RKSJ-H"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3(TrueType) */
       local("YuGothic-Medium"), local("YuGothic"), /* \u6E38\u30B4\u30B7\u30C3\u30AF\u4F53(macOS) */
       local("Yu Gothic Medium"), local("Yu Gothic"), /* \u6E38\u30B4\u30B7\u30C3\u30AF(Windows) "Yu Gothic" is a fallback. */
       local("BIZ UDPGothic"); /* BIZ UDP\u30B4\u30B7\u30C3\u30AF */
}

/* \u7E26\u7D44\u307F\u7528 (vertical writing) */

@font-face {
  font-family: "serif-ja-v";
  src: local("\uFF2D\uFF33 \u660E\u671D"), /* for IE */
       local("MS Mincho"), /* \uFF2D\uFF33 \u660E\u671D */
       local("HiraMinProN-W3"), local("Hiragino Mincho ProN"), /* \u30D2\u30E9\u30AE\u30CE\u660E\u671D ProN W3 */
       local("HiraMinPro-W3"), local("Hiragino Mincho Pro"), /* \u30D2\u30E9\u30AE\u30CE\u660E\u671D Pro W3 */
       local("YuMin-Medium"), local("YuMincho"), /* \u6E38\u660E\u671D\u4F53(macOS) */
       local("Yu Mincho"), /* \u6E38\u660E\u671D(Windows) */
       local("BIZ UDMincho"); /*  BIZ UD\u660E\u671D */
}

@font-face {
  font-family: "sans-serif-ja-v";
  src: local("\uFF2D\uFF33 \u30B4\u30B7\u30C3\u30AF"), /* for IE */
       local("MS Gothic"), /* \uFF2D\uFF33 \u30B4\u30B7\u30C3\u30AF */
       local("HiraginoSans-W3"), local("Hiragino Sans"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4\u30B7\u30C3\u30AF */
       local("HiraKakuProN-W3"), local("Hiragino Kaku Gothic ProN"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 ProN W3 */
       local("HiraKakuPro-W3"), local("Hiragino Kaku Gothic Pro"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 Pro W3 */
       local("\u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3"), /* for old Safari */
       local("HiraKakuDS-W3-83pv-RKSJ-H"), /* \u30D2\u30E9\u30AE\u30CE\u89D2\u30B4 W3(TrueType) */
       local("YuGothic-Medium"), local("YuGothic"), /* \u6E38\u30B4\u30B7\u30C3\u30AF\u4F53(macOS) */
       local("Yu Gothic Medium"), local("Yu Gothic"), /* \u6E38\u30B4\u30B7\u30C3\u30AF(Windows)  "Yu Gothic" is a fallback. */
       local("BIZ UDGothic"); /* BIZ UD\u30B4\u30B7\u30C3\u30AF */
}`});var Oe=class{close(){}links(){return[]}get(t){throw Error("This is an empty fetcher")}};var Me=class{constructor(t,e){this.client=t||window.fetch.bind(window),this.baseUrl=e}links(){return[]}get(t){let e=t.toURL(this.baseUrl);if(e===void 0)throw Error(`Invalid HREF: ${t.href}`);return new Ne(this.client,t,e)}close(){}},Ne=class{constructor(t,e,i){this.client=t||window.fetch.bind(window),this._link=e,this.url=i}async headResponse(){if(this._headResponse)return this._headResponse;let t=await this.client(this.url,{method:"HEAD"});if(!t.ok)throw new Error(`http HEAD request for ${this.url} failed with HTTP status code ${t.status}`);return this._headResponse=t,t}close(){}async link(){return this._link}async read(t,e){if(t)throw new Error("http read range not implemented!");let i=await this.client(this.url,e?.signal?{signal:e.signal}:void 0);if(!i.ok)throw new Error(`http GET request for ${this.url} failed with HTTP status code ${i.status}`);return new Uint8Array(await i.arrayBuffer())}async length(){let t=(await this.headResponse()).headers.get("content-length");if(t===null||t==="")throw new Error("length for resource unavailable");return parseInt(t)}async readAsJSON(){let t=await this.client(this.url);if(!t.ok)throw new Error(`http GET request for ${this.url} failed with HTTP status code ${t.status}`);return await t.json()}async readAsString(){let t=await this.client(this.url);if(!t.ok)throw new Error(`http GET request for ${this.url} failed with HTTP status code ${t.status}`);return await t.text()}async readAsXML(){let t=await this.client(this.url);if(!t.ok)throw new Error(`http GET request for ${this.url} failed with HTTP status code ${t.status}`);return new DOMParser().parseFromString(await t.text(),this._link.mediaType.isHTML?this._link.mediaType.essence:"text/xml")}};function Hi(r){return r&&Array.isArray(r)?r:void 0}function A(r){return r&&typeof r=="string"?[r]:Hi(r)}function Wi(r){return typeof r=="string"?new Date(r):void 0}function zt(r){return isNaN(r)?void 0:r}function rt(r){return zt(r)!==void 0&&Math.sign(r)>=0?r:void 0}function Lt(r){let t=new Array;return r.forEach(e=>t.push(e)),t}var Ae=class r{constructor(t={}){this.conformsTo=t.conformsTo??[],this.certification=t.certification??null,this.summary=t.summary??null,this.accessMode=t.accessMode??[],this.accessModeSufficient=t.accessModeSufficient??[],this.feature=t.feature??[],this.hazard=t.hazard??[],this.exemption=t.exemption??[]}static deserialize(t){if(!t||typeof t!="object")return;let e=t,i=A(e.conformsTo),o=A(e.accessMode),n=Hi(e.accessModeSufficient),s=A(e.feature),a=A(e.hazard),h=A(e.exemption);return new r({conformsTo:i?i.map(l=>Cs.deserialize(l)).filter(l=>l!==void 0):void 0,certification:e.certification?Bi.deserialize(e.certification):void 0,summary:e.summary,accessMode:o?o.map(l=>Es.deserialize(l)).filter(l=>l!==void 0):void 0,accessModeSufficient:n?n.map(l=>Ps.deserialize(l)).filter(l=>l!==void 0):void 0,feature:s?s.map(l=>Fs.deserialize(l)).filter(l=>l!==void 0):void 0,hazard:a?a.map(l=>xs.deserialize(l)).filter(l=>l!==void 0):void 0,exemption:h?h.map(l=>ks.deserialize(l)).filter(l=>l!==void 0):void 0})}serialize(){let t={};return this.conformsTo?.length>0&&(t.conformsTo=this.conformsTo.map(e=>e.serialize())),this.certification!==void 0&&this.certification!==null&&(t.certification=this.certification.serialize()),this.summary!==void 0&&this.summary!==null&&(t.summary=this.summary),this.accessMode?.length>0&&(t.accessMode=this.accessMode.map(e=>e.serialize())),this.accessModeSufficient?.length>0&&(t.accessModeSufficient=this.accessModeSufficient.map(e=>e.serialize())),this.feature?.length>0&&(t.feature=this.feature.map(e=>e.serialize())),this.hazard?.length>0&&(t.hazard=this.hazard.map(e=>e.serialize())),this.exemption?.length>0&&(t.exemption=this.exemption.map(e=>e.serialize())),t}},x=class J{constructor(t){this.uri=t}static deserialize(t){if(!(!t||typeof t!="string"))return new J(t)}serialize(){return this.uri}get isWCAGLevelA(){return this===J.EPUB_A11Y_10_WCAG_20_A||this===J.EPUB_A11Y_11_WCAG_20_A||this===J.EPUB_A11Y_11_WCAG_21_A||this===J.EPUB_A11Y_11_WCAG_22_A}get isWCAGLevelAA(){return this===J.EPUB_A11Y_10_WCAG_20_AA||this===J.EPUB_A11Y_11_WCAG_20_AA||this===J.EPUB_A11Y_11_WCAG_21_AA||this===J.EPUB_A11Y_11_WCAG_22_AA}get isWCAGLevelAAA(){return this===J.EPUB_A11Y_10_WCAG_20_AAA||this===J.EPUB_A11Y_11_WCAG_20_AAA||this===J.EPUB_A11Y_11_WCAG_21_AAA||this===J.EPUB_A11Y_11_WCAG_22_AAA}};x.EPUB_A11Y_10_WCAG_20_A=new x("http://www.idpf.org/epub/a11y/accessibility-20170105.html#wcag-a"),x.EPUB_A11Y_10_WCAG_20_AA=new x("http://www.idpf.org/epub/a11y/accessibility-20170105.html#wcag-aa"),x.EPUB_A11Y_10_WCAG_20_AAA=new x("http://www.idpf.org/epub/a11y/accessibility-20170105.html#wcag-aaa"),x.EPUB_A11Y_11_WCAG_20_A=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.0-a"),x.EPUB_A11Y_11_WCAG_20_AA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.0-aa"),x.EPUB_A11Y_11_WCAG_20_AAA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.0-aaa"),x.EPUB_A11Y_11_WCAG_21_A=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.1-a"),x.EPUB_A11Y_11_WCAG_21_AA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.1-aa"),x.EPUB_A11Y_11_WCAG_21_AAA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.1-aaa"),x.EPUB_A11Y_11_WCAG_22_A=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.2-a"),x.EPUB_A11Y_11_WCAG_22_AA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.2-aa"),x.EPUB_A11Y_11_WCAG_22_AAA=new x("https://www.w3.org/TR/epub-a11y-11#wcag-2.2-aaa");var Cs=x,Bi=class r{constructor(t=null,e=null,i=null){this.certifiedBy=t,this.credential=e,this.report=i}static deserialize(t){if(!(!t||typeof t!="object"))return new r(t.certifiedBy,t.credential,t.report)}serialize(){let t={};return this.certifiedBy&&(t.certifiedBy=this.certifiedBy),this.credential&&(t.credential=this.credential),this.report&&(t.report=this.report),t}},k=class Bo{constructor(t){this.value=t}static deserialize(t){if(!(!t||typeof t!="string"))return new Bo(t)}serialize(){return this.value}};k.AUDITORY=new k("auditory"),k.CHART_ON_VISUAL=new k("chartOnVisual"),k.CHEM_ON_VISUAL=new k("chemOnVisual"),k.COLOR_DEPENDENT=new k("colorDependent"),k.DIAGRAM_ON_VISUAL=new k("diagramOnVisual"),k.MATH_ON_VISUAL=new k("mathOnVisual"),k.MUSIC_ON_VISUAL=new k("musicOnVisual"),k.TACTILE=new k("tactile"),k.TEXT_ON_VISUAL=new k("textOnVisual"),k.TEXTUAL=new k("textual"),k.VISUAL=new k("visual");var Es=k,ut=class Ot{constructor(t){if(typeof t=="string"){if(!Ot.VALID_MODES.has(t.toLowerCase()))return;this.value=t.toLowerCase()}else{let e=t.filter(i=>Ot.VALID_MODES.has(i.toLowerCase()));if(e.length===0)return;this.value=Array.from(new Set(e))}}static deserialize(t){if(!t)return;if(typeof t=="string")return Ot.VALID_MODES.has(t.toLowerCase())?new Ot(t):void 0;if(!Array.isArray(t))return;let e=t.filter(i=>i?Ot.VALID_MODES.has(i.toLowerCase()):!1);if(e.length!==0)return new Ot(e)}serialize(){return this.value}};ut.VALID_MODES=new Set(["auditory","tactile","textual","visual"]),ut.AUDITORY=new ut("auditory"),ut.TACTILE=new ut("tactile"),ut.TEXTUAL=new ut("textual"),ut.VISUAL=new ut("visual");var Ps=ut,y=class jo{constructor(t){this.value=t}static deserialize(t){if(!(!t||typeof t!="string"))return new jo(t)}serialize(){return this.value}};y.NONE=new y("none"),y.ANNOTATIONS=new y("annotations"),y.ARIA=new y("ARIA"),y.INDEX=new y("index"),y.PAGE_BREAK_MARKERS=new y("pageBreakMarkers"),y.PAGE_NAVIGATION=new y("pageNavigation"),y.PRINT_PAGE_NUMBERS=new y("printPageNumbers"),y.READING_ORDER=new y("readingOrder"),y.STRUCTURAL_NAVIGATION=new y("structuralNavigation"),y.TABLE_OF_CONTENTS=new y("tableOfContents"),y.TAGGED_PDF=new y("taggedPDF"),y.ALTERNATIVE_TEXT=new y("alternativeText"),y.AUDIO_DESCRIPTION=new y("audioDescription"),y.CAPTIONS=new y("captions"),y.CLOSED_CAPTIONS=new y("closedCaptions"),y.DESCRIBED_MATH=new y("describedMath"),y.LONG_DESCRIPTION=new y("longDescription"),y.OPEN_CAPTIONS=new y("openCaptions"),y.SIGN_LANGUAGE=new y("signLanguage"),y.TRANSCRIPT=new y("transcript"),y.DISPLAY_TRANSFORMABILITY=new y("displayTransformability"),y.SYNCHRONIZED_AUDIO_TEXT=new y("synchronizedAudioText"),y.TIMING_CONTROL=new y("timingControl"),y.UNLOCKED=new y("unlocked"),y.CHEM_ML=new y("ChemML"),y.LATEX=new y("latex"),y.LATEX_CHEMISTRY=new y("latex-chemistry"),y.MATH_ML=new y("MathML"),y.MATH_ML_CHEMISTRY=new y("MathML-chemistry"),y.TTS_MARKUP=new y("ttsMarkup"),y.HIGH_CONTRAST_AUDIO=new y("highContrastAudio"),y.HIGH_CONTRAST_DISPLAY=new y("highContrastDisplay"),y.LARGE_PRINT=new y("largePrint"),y.BRAILLE=new y("braille"),y.TACTILE_GRAPHIC=new y("tactileGraphic"),y.TACTILE_OBJECT=new y("tactileObject"),y.FULL_RUBY_ANNOTATIONS=new y("fullRubyAnnotations"),y.HORIZONTAL_WRITING=new y("horizontalWriting"),y.RUBY_ANNOTATIONS=new y("rubyAnnotations"),y.VERTICAL_WRITING=new y("verticalWriting"),y.WITH_ADDITIONAL_WORD_SEGMENTATION=new y("withAdditionalWordSegmentation"),y.WITHOUT_ADDITIONAL_WORD_SEGMENTATION=new y("withoutAdditionalWordSegmentation");var Fs=y,T=class Go{constructor(t){this.value=t}static deserialize(t){if(!(!t||typeof t!="string"))return new Go(t)}serialize(){return this.value}};T.FLASHING=new T("flashing"),T.NO_FLASHING_HAZARD=new T("noFlashingHazard"),T.UNKNOWN_FLASHING_HAZARD=new T("unknownFlashingHazard"),T.MOTION_SIMULATION=new T("motionSimulation"),T.NO_MOTION_SIMULATION_HAZARD=new T("noMotionSimulationHazard"),T.UNKNOWN_MOTION_SIMULATION_HAZARD=new T("unknownMotionSimulationHazard"),T.SOUND=new T("sound"),T.NO_SOUND_HAZARD=new T("noSoundHazard"),T.UNKNOWN_SOUND_HAZARD=new T("unknownSoundHazard"),T.UNKNOWN=new T("unknown"),T.NONE=new T("none");var xs=T,O=class Vo{constructor(t){this.value=t}static deserialize(t){if(!(!t||typeof t!="string"))return new Vo(t)}serialize(){return this.value}};O.NONE=new O("none"),O.DOCUMENTED=new O("documented"),O.LEGAL=new O("legal"),O.TEMPORARY=new O("temporary"),O.TECHNICAL=new O("technical"),O.EAA_DISPROPORTIONATE_BURDEN=new O("eaa-disproportionate-burden"),O.EAA_FUNDAMENTAL_ALTERATION=new O("eaa-fundamental-alteration"),O.EAA_MICROENTERPRISE=new O("eaa-microenterprise"),O.EAA_TECHNICAL_IMPOSSIBILITY=new O("eaa-technical-impossibility"),O.EAA_TEMPORARY=new O("eaa-temporary");var ks=O;var _=(r=>(r.reflowable="reflowable",r.fixed="fixed",r.scrolled="scrolled",r))(_||{});var ge=class r{constructor(t){this.cssSelector=t.cssSelector,this.textNodeIndex=t.textNodeIndex,this.charOffset=t.charOffset}static deserialize(t){if(!(t&&t.cssSelector))return;let e=rt(t.textNodeIndex);if(e===void 0)return;let i=rt(t.charOffset);return i===void 0&&(i=rt(t.offset)),new r({cssSelector:t.cssSelector,textNodeIndex:e,charOffset:i})}serialize(){let t={cssSelector:this.cssSelector,textNodeIndex:this.textNodeIndex};return this.charOffset!==void 0&&(t.charOffset=this.charOffset),t}};var De=class r{constructor(t){this.start=t.start,this.end=t.end}static deserialize(t){if(!t)return;let e=ge.deserialize(t.start);if(e)return new r({start:e,end:ge.deserialize(t.end)})}serialize(){let t={start:this.start.serialize()};return this.end&&(t.end=this.end.serialize()),t}};function Mt(r){let t=r.split(",")[0].trim(),e=(t.toLowerCase().startsWith("npt:")?t.slice(4):t).split(":");if(e.length===1){let i=parseFloat(e[0]);return isNaN(i)?void 0:i}if(e.length===2){let i=parseInt(e[0],10),o=parseFloat(e[1]);return isNaN(i)||isNaN(o)?void 0:i*60+o}if(e.length===3){let i=parseInt(e[0],10),o=parseInt(e[1],10),n=parseFloat(e[2]);return isNaN(i)||isNaN(o)||isNaN(n)?void 0:i*3600+o*60+n}}function Xo(r){let t=Mt(r);return t!==void 0&&t===0}function $o(r){let t=Math.floor(r),e=Math.floor(t/3600),i=Math.floor(t%3600/60),o=t%60,n=String(i).padStart(2,"0"),s=String(o).padStart(2,"0");return e>0?`${e}:${n}:${s}`:`${i}:${s}`}var Ue=null,ji=null,fe=0,Qt={r:255,g:255,b:255,a:1},Zt=new Map,Ts=()=>{if(!Ue)if(typeof OffscreenCanvas<"u")Ue=new OffscreenCanvas(5,5),ji=Ue.getContext("2d",{willReadFrequently:!0,desynchronized:!0});else{let r=document.createElement("canvas");r.width=5,r.height=5,Ue=r,ji=r.getContext("2d",{willReadFrequently:!0,desynchronized:!0})}return ji},zs=r=>{if(!r)return!0;let t=r.trim().toLowerCase();return t.startsWith("var(")||["transparent","currentcolor","inherit","initial","revert","unset","revert-layer"].includes(t)?!0:["linear-gradient","radial-gradient","conic-gradient","repeating-linear-gradient","repeating-radial-gradient","repeating-conic-gradient"].some(e=>t.includes(e))},Ie=(r,t)=>{console.warn(`[@readium/helpers] Could not parse color: "${r}". ${t} Falling back to ${JSON.stringify(Qt)} to compute contrast. Please use a CSS color value that can be computed to RGB(A).`)},ft=(r,t=null)=>{let e=t?`${r}|${t}`:r,i=Zt.get(e);if(i!==void 0)return i??Qt;if(zs(r))return Ie(r,"Unsupported color format or special value."),Zt.set(e,null),Qt;let o=Ts();if(!o)return Ie(r,"Could not get canvas context."),Zt.set(e,null),Qt;try{fe===0&&o.clearRect(0,0,5,5);let n=fe%5,s=Math.floor(fe/5);o.clearRect(n,s,1,1),t&&(o.fillStyle=t,o.fillRect(n,s,1,1)),o.fillStyle=r,o.fillRect(n,s,1,1);let a=o.getImageData(n,s,1,1);fe=(fe+1)%25;let[h,l,d,c]=a.data;if(c===0)return Ie(r,"Fully transparent color."),Zt.set(e,null),Qt;let m={r:h,g:l,b:d,a:c/255};return Zt.set(e,m),m}catch(n){return Ie(r,`Error: ${n instanceof Error?n.message:String(n)}`),Zt.set(e,null),Qt}},Gi=r=>{let t=r/255;return t<=.03928?t/12.92:Math.pow((t+.055)/1.055,2.4)},He=r=>{let t=Gi(r.r),e=Gi(r.g),i=Gi(r.b);return .2126*t+.7152*e+.0722*i},Se=(r,t)=>{let e=typeof r=="string"?ft(r):r,i=typeof t=="string"?ft(t):t,o=He(e),n=He(i),s=Math.max(o,n),a=Math.min(o,n);return(s+.05)/(a+.05)},We=(r,t=null)=>{let e=ft(r,t),i=Se(e,{r:255,g:255,b:255,a:1}),o=Se(e,{r:0,g:0,b:0,a:1});return i>o};var Be=(r,t=null)=>We(r,t)?"white":"black",Ls=r=>{let t=r.a!==void 0?r.a:1;return`rgba(${Math.round(r.r)}, ${Math.round(r.g)}, ${Math.round(r.b)}, ${t})`},Os=(r,t)=>({r:Math.min(255,r.r+(255-r.r)*t),g:Math.min(255,r.g+(255-r.g)*t),b:Math.min(255,r.b+(255-r.b)*t),a:r.a??1}),Ms=(r,t)=>({r:Math.max(0,r.r*(1-t)),g:Math.max(0,r.g*(1-t)),b:Math.max(0,r.b*(1-t)),a:r.a??1}),St=(r,t=null,e=3)=>{let i=ft(r),o=t?ft(t):{r:255,g:255,b:255,a:1},n=Se(i,o);if(n>=e)return r;let s=He(o)<.5,a={...i,a:i.a??1},h=20,l=.1;for(let d=0;d<h&&(s?a=Os(a,l):a=Ms(a,l),n=Se(a,o),!(n>=e));d++);return Ls(a)};var Ko=()=>typeof navigator>"u"?"":navigator.userAgent||"",Yo=()=>typeof navigator>"u"?void 0:navigator.userAgentData||void 0,je=class{constructor(){let t=Yo(),e=Ko(),i=n=>(typeof n=="string"||typeof n=="number")&&n?String(n).replace(/_/g,".").split(".").map(s=>parseInt(s)||0):[],o=(n="")=>{if(!n)return[];let s=new RegExp("^.*"+n+"[ :\\/]?(\\d+([\\._]\\d+)*).*$");return s.test(e)?i(e.replace(s,"$1")):[]};this.OS=(n=>(/(macOS|Mac OS X)/.test(e)?(/\(iP(hone|od touch);/.test(e)&&(n.iOS=o("CPU (?:iPhone )?OS ")),/\(iPad;/.test(e)?n.iOS=n.iPadOS=o("CPU (?:iPhone )?OS "):/(macOS|Mac OS X) \d/.test(e)&&(document.ontouchend!==void 0?n.iOS=n.iPadOS=o():n.macOS=o("(?:macOS|Mac OS X) "))):/Windows( NT)? \d/.test(e)?n.Windows=(s=>s[0]!==6||!s[1]?s:s[1]===1?[7]:s[1]===2?[8]:[8,1])(o("Windows(?: NT)?")):/Android \d/.test(e)?n.Android=o("Android"):/CrOS/.test(e)?n.ChromeOS=o():/X11;/.test(e)&&(n.Linux=o()),n))({}),t&&t.getHighEntropyValues(["architecture","model","platform","platformVersion","uaFullVersion"]).then(n=>(s=>{let a=n.platform,h=n.platformVersion;if(!(!a||!h)){if(/^i(OS|P(hone|od touch))$/.test(a))s.iOS=i(h);else if(/^iPad(OS)?$/.test(a))s.iOS=s.iPadOS=i(h);else if(/^(macOS|(Mac )?OS X|Mac(Intel)?)$/.test(a))document.ontouchend!==void 0?s.iOS=s.iPadOS=i():s.macOS=i(h);else if(/^(Microsoft )?Windows$/.test(a))s.Windows=i(h);else if(/^(Google )?Android$/.test(a))s.Android=i(h);else if(/^((Google )?Chrome OS|CrOS)$/.test(a))s.ChromeOS=i(h);else if(/^(Linux|Ubuntu|X11)$/.test(a))s.Linux=i(h);else return;Object.keys(this.OS).forEach(l=>delete this.OS[l]),Object.assign(this.OS,s)}})({})),this.UA=(n=>{let s=!1;if(t&&Array.isArray(t.brands)){let a=t.brands.reduce((h,l)=>(h[l.brand]=[l.version*1],h),{});a["Google Chrome"]?(s=!0,n.Blink=n.Chromium=a.Chromium||[],n.Chrome=a["Google Chrome"]):a["Microsoft Edge"]?(s=!0,n.Blink=n.Chromium=a.Chromium||[],n.Edge=a["Microsoft Edge"]):a.Opera&&(s=!0,n.Blink=n.Chromium=a.Chromium||[],n.Opera=a.Opera)}return s||(/ Gecko\/\d/.test(e)?(n.Gecko=o("rv"),/ Waterfox\/\d/.test(e)?n.Waterfox=o("Waterfox"):/ Firefox\/\d/.test(e)&&(n.Firefox=o("Firefox"))):/ Edge\/\d/.test(e)?(n.EdgeHTML=o("Edge"),n.Edge=n.EdgeHTML):/ Chrom(ium|e)\/\d/.test(e)?(n.Blink=n.Chromium=(a=>a[0]?a:o("Chrome"))(o("Chromium")),/ EdgA?\/\d/.test(e)?n.Edge=(a=>a[0]?a:o("Edg"))(o("EdgA")):/ OPR\/\d/.test(e)?n.Opera=o("OPR"):/ Vivaldi\/\d/.test(e)?n.Vivaldi=o("Vivaldi"):/ Silk\/\d/.test(e)?n.Silk=o("Silk"):/ UCBrowser\/\d/.test(e)?n.UCBrowser=o("UCBrowser"):/ Phoebe\/\d/.test(e)?n.Phoebe=o("Phoebe"):n.Chrome=(a=>a[0]?a:n.Chromium)(o("Chrome"))):/ AppleWebKit\/\d/.test(e)?(n.WebKit=o("AppleWebKit"),/ CriOS \d/.test(e)?n.Chrome=o("CriOS"):/ FxiOS \d/.test(e)?n.Firefox=o("FxiOS"):/ EdgiOS\/\d/.test(e)?n.Edge=o("EdgiOS"):/ Version\/\d/.test(e)&&(n.Safari=o("Version"))):/ Trident\/\d/.test(e)&&(n.Trident=o("Trident"),n.InternetExplorer=(a=>a[0]?a:o("MSIE"))(o("rv")))),/[\[; ]FB(AN|_IAB)\//.test(e)&&(n.Facebook=o("FBAV")),/ Line\/\d/.test(e)&&(n.LINE=o("Line")),n})({}),this.Env={get:()=>[this.OS,this.UA].reduce((n,s)=>{for(let a in s)s[a]&&n.push(a);return n},[])}}},Vi=class extends je{get iOSRequest(){let t=Yo(),e=Ko();if(this.OS.iOS&&!this.OS.iPadOS)return"mobile";if(this.OS.iPadOS)return/\(iPad;/.test(e)||t&&/^iPad(OS)?$/.test(t.platform)?"mobile":"desktop"}},j=new je,st=new Vi;var qo=":~:text=";function Xi(r){if(!r)return;let t=r.indexOf(qo);if(t===-1)return;let e=r.slice(t+qo.length).split("&")[0];if(e)try{let i=e.split(","),o={textStart:""};return i.length>1&&i[0].endsWith("-")&&(o.prefix=decodeURIComponent(i[0].slice(0,-1)),i=i.slice(1)),i.length>1&&i[i.length-1].startsWith("-")&&(o.suffix=decodeURIComponent(i[i.length-1].slice(1)),i=i.slice(0,-1)),i.length===0||i[0]===""?void 0:(o.textStart=decodeURIComponent(i[0]),i.length>1&&(o.textEnd=decodeURIComponent(i[1])),o)}catch{return}}var Ns=["ADDRESS","ARTICLE","ASIDE","BLOCKQUOTE","BR","DETAILS","DIALOG","DD","DIV","DL","DT","FIELDSET","FIGCAPTION","FIGURE","FOOTER","FORM","H1","H2","H3","H4","H5","H6","HEADER","HGROUP","HR","LI","MAIN","NAV","OL","P","PRE","SECTION","TABLE","UL","TR","TH","TD","COLGROUP","COL","CAPTION","THEAD","TBODY","TFOOT"],Ge=/[\t-\r -#%-\*,-\/:;\?@\[-\]_\{\}\x85\xA0\xA1\xA7\xAB\xB6\xB7\xBB\xBF\u037E\u0387\u055A-\u055F\u0589\u058A\u05BE\u05C0\u05C3\u05C6\u05F3\u05F4\u0609\u060A\u060C\u060D\u061B\u061E\u061F\u066A-\u066D\u06D4\u0700-\u070D\u07F7-\u07F9\u0830-\u083E\u085E\u0964\u0965\u0970\u0AF0\u0DF4\u0E4F\u0E5A\u0E5B\u0F04-\u0F12\u0F14\u0F3A-\u0F3D\u0F85\u0FD0-\u0FD4\u0FD9\u0FDA\u104A-\u104F\u10FB\u1360-\u1368\u1400\u166D\u166E\u1680\u169B\u169C\u16EB-\u16ED\u1735\u1736\u17D4-\u17D6\u17D8-\u17DA\u1800-\u180A\u1944\u1945\u1A1E\u1A1F\u1AA0-\u1AA6\u1AA8-\u1AAD\u1B5A-\u1B60\u1BFC-\u1BFF\u1C3B-\u1C3F\u1C7E\u1C7F\u1CC0-\u1CC7\u1CD3\u2000-\u200A\u2010-\u2029\u202F-\u2043\u2045-\u2051\u2053-\u205F\u207D\u207E\u208D\u208E\u2308-\u230B\u2329\u232A\u2768-\u2775\u27C5\u27C6\u27E6-\u27EF\u2983-\u2998\u29D8-\u29DB\u29FC\u29FD\u2CF9-\u2CFC\u2CFE\u2CFF\u2D70\u2E00-\u2E2E\u2E30-\u2E44\u3000-\u3003\u3008-\u3011\u3014-\u301F\u3030\u303D\u30A0\u30FB\uA4FE\uA4FF\uA60D-\uA60F\uA673\uA67E\uA6F2-\uA6F7\uA874-\uA877\uA8CE\uA8CF\uA8F8-\uA8FA\uA8FC\uA92E\uA92F\uA95F\uA9C1-\uA9CD\uA9DE\uA9DF\uAA5C-\uAA5F\uAADE\uAADF\uAAF0\uAAF1\uABEB\uFD3E\uFD3F\uFE10-\uFE19\uFE30-\uFE52\uFE54-\uFE61\uFE63\uFE68\uFE6A\uFE6B\uFF01-\uFF03\uFF05-\uFF0A\uFF0C-\uFF0F\uFF1A\uFF1B\uFF1F\uFF20\uFF3B-\uFF3D\uFF3F\uFF5B\uFF5D\uFF5F-\uFF65]|\uD800[\uDD00-\uDD02\uDF9F\uDFD0]|\uD801\uDD6F|\uD802[\uDC57\uDD1F\uDD3F\uDE50-\uDE58\uDE7F\uDEF0-\uDEF6\uDF39-\uDF3F\uDF99-\uDF9C]|\uD804[\uDC47-\uDC4D\uDCBB\uDCBC\uDCBE-\uDCC1\uDD40-\uDD43\uDD74\uDD75\uDDC5-\uDDC9\uDDCD\uDDDB\uDDDD-\uDDDF\uDE38-\uDE3D\uDEA9]|\uD805[\uDC4B-\uDC4F\uDC5B\uDC5D\uDCC6\uDDC1-\uDDD7\uDE41-\uDE43\uDE60-\uDE6C\uDF3C-\uDF3E]|\uD807[\uDC41-\uDC45\uDC70\uDC71]|\uD809[\uDC70-\uDC74]|\uD81A[\uDE6E\uDE6F\uDEF5\uDF37-\uDF3B\uDF44]|\uD82F\uDC9F|\uD836[\uDE87-\uDE8B]|\uD83A[\uDD5E\uDD5F]/u;var $i=(r,t,e)=>{let i=[],o=t.createRange();for(o.selectNodeContents(e??t);!o.collapsed&&i.length<2;){let n;if(r.prefix){let s=ye(r.prefix,o);if(s==null)break;Ve(o,s.startContainer,s.startOffset);let a=t.createRange();if(a.setStart(s.endContainer,s.endOffset),a.setEnd(o.endContainer,o.endOffset),Qo(a),a.collapsed||(n=ye(r.textStart,a),n==null))break;if(n.compareBoundaryPoints(Range.START_TO_START,a)!==0)continue}else{if(n=ye(r.textStart,o),n==null)break;Ve(o,n.startContainer,n.startOffset)}if(r.textEnd){let s=t.createRange();s.setStart(n.endContainer,n.endOffset),s.setEnd(o.endContainer,o.endOffset);let a=!1;for(;!s.collapsed&&i.length<2;){let h=ye(r.textEnd,s);if(h==null)break;if(Ve(s,h.startContainer,h.startOffset),n.setEnd(h.endContainer,h.endOffset),r.suffix){let l=Jo(r.suffix,n,o,t);if(l===yt.NO_SUFFIX_MATCH)break;if(l===yt.SUFFIX_MATCH){a=!0,i.push(n.cloneRange());continue}else if(l===yt.MISPLACED_SUFFIX)continue}else a=!0,i.push(n.cloneRange())}if(!a)break}else if(r.suffix){let s=Jo(r.suffix,n,o,t);if(s===yt.NO_SUFFIX_MATCH)break;if(s===yt.SUFFIX_MATCH){i.push(n.cloneRange()),Ve(o,o.startContainer,o.startOffset);continue}else if(s===yt.MISPLACED_SUFFIX)continue}else i.push(n.cloneRange())}return i},yt={NO_SUFFIX_MATCH:0,SUFFIX_MATCH:1,MISPLACED_SUFFIX:2},Jo=(r,t,e,i)=>{let o=i.createRange();o.setStart(t.endContainer,t.endOffset),o.setEnd(e.endContainer,e.endOffset),Qo(o);let n=ye(r,o);return n==null?yt.NO_SUFFIX_MATCH:n.compareBoundaryPoints(Range.START_TO_START,o)!==0?yt.MISPLACED_SUFFIX:yt.SUFFIX_MATCH},Ve=(r,t,e)=>{try{r.setStart(t,e+1)}catch{r.setStartAfter(t)}},Qo=r=>{let t=As(r),e=t.nextNode();for(;!r.collapsed&&e!=null;){if(e!==r.startContainer&&r.setStart(e,0),e.textContent.length>r.startOffset&&!e.textContent[r.startOffset].match(/\s/))return;try{r.setStart(e,r.startOffset+1)}catch{e=t.nextNode(),e==null?r.collapse():r.setStart(e,0)}}},As=r=>(r.commonAncestorContainer.ownerDocument??r.commonAncestorContainer).createTreeWalker(r.commonAncestorContainer,NodeFilter.SHOW_TEXT|NodeFilter.SHOW_ELEMENT,{acceptNode:t=>Is(t,r)}),Ds=r=>{if(r.hidden==="until-found")return!0;let t=r.attributes;return!!(t&&t.hidden&&t.hidden.value==="until-found")},tn=r=>{let t=r;for(;t!=null&&t.nodeType!==Node.ELEMENT_NODE;)t=t.parentNode;let e=t,i=e?.ownerDocument?.defaultView;if(e!=null&&i!=null){if(Ds(e))return!0;let o=i.getComputedStyle(e);if(o.visibility==="hidden"||o.display==="none"||parseInt(o.height,10)===0&&o.overflowY!="visible"||parseInt(o.width,10)===0&&o.overflowX!="visible"||parseInt(o.opacity,10)===0)return!1}return!0},Us=(r,t)=>t!=null&&!t.intersectsNode(r)?NodeFilter.FILTER_REJECT:tn(r)?NodeFilter.FILTER_ACCEPT:NodeFilter.FILTER_REJECT,Is=(r,t)=>t!=null&&!t.intersectsNode(r)||!tn(r)?NodeFilter.FILTER_REJECT:r.nodeType===Node.TEXT_NODE?NodeFilter.FILTER_ACCEPT:NodeFilter.FILTER_SKIP,Hs=(r,t)=>{let e=[],i=[],o=Array.from(Bs(r,n=>Us(n,t)));for(let n of o)n.nodeType===Node.TEXT_NODE?i.push(n):n.nodeType===Node.ELEMENT_NODE&&Ns.includes(n.tagName.toUpperCase())&&i.length>0&&(e.push(i),i=[]);return i.length>0&&e.push(i),e},Ws=(r,t,e)=>{let i="";return r.length===1?i=r[0].textContent.substring(t,e):i=r[0].textContent.substring(t)+r.slice(1,-1).reduce((o,n)=>o+n.textContent,"")+r.slice(-1)[0].textContent.substring(0,e),i.replace(/[\t\n\r ]+/g," ")};function*Bs(r,t){let e=(r.ownerDocument??r).createTreeWalker(r,NodeFilter.SHOW_ELEMENT|NodeFilter.SHOW_TEXT,{acceptNode:t}),i=new Set;for(;Xs(e,i)!==null;)yield e.currentNode}var ye=(r,t)=>{let e=Hs(t.commonAncestorContainer,t),i=Vs(t.commonAncestorContainer.ownerDocument??void 0);for(let o of e){let n=js(r,t,o,i);if(n!==void 0)return n}},js=(r,t,e,i)=>{if(!r||!t||!(e||[]).length)return;let o=e[0]===t.startContainer?t.startOffset:0,n=Nt(Ws(e,o,void 0)),s=Nt(r),a=0,h,l;for(;a<n.length;){let d=n.indexOf(s,a);if(d===-1)return;if(Gs(n,d,s.length,i)){let c=Nt(e[0].data.slice(0,o)).length;h=Zo(c+d,e,!1),l=Zo(c+d+s.length,e,!0)}if(h!=null&&l!=null){let c=new Range;if(c.setStart(h.node,h.offset),c.setEnd(l.node,l.offset),t.compareBoundaryPoints(Range.START_TO_START,c)<=0&&t.compareBoundaryPoints(Range.END_TO_END,c)>=0)return c}a=d+1}},Zo=(r,t,e)=>{let i=0,o;for(let n=0;n<t.length;n++){let s=t[n];o||(o=Nt(s.data));let a=i+o.length;if(e&&(a+=1),a>r){let h=r-i,l=Math.min(r-i,s.data.length),d=e?o.substring(0,h):o.substring(h),c=Nt(e?s.data.substring(0,l):s.data.substring(l)),m=(e?-1:1)*(d.length>c.length?-1:1);for(;l>=0&&l<=s.data.length;){if(c.length===d.length)return{node:s,offset:l};l+=m,c=Nt(e?s.data.substring(0,l):s.data.substring(l))}}if(i+=o.length,n+1<t.length){let h=Nt(t[n+1].data);o.slice(-1)===" "&&h.slice(0,1)===" "&&(i-=1),o=h}}},Gs=(r,t,e,i)=>{if(t<0||t>=r.length||e<=0||t+e>r.length)return!1;if(i){let o=i.segment(r),n=o.containing(t);if(!n||n.isWordLike&&n.index!=t)return!1;let s=t+e,a=o.containing(s);if(a&&a.isWordLike&&a.index!=s)return!1}else if(r[t].match(Ge)&&(++t,--e,!e)||r[t+e-1].match(Ge)&&(--e,!e)||t!==0&&!r[t-1].match(Ge)||t+e!==r.length&&!r[t+e].match(Ge))return!1;return!0},Nt=r=>(r||"").normalize("NFKD").replace(/\s+/g," ").replace(/[\u0300-\u036f]/g,"").toLowerCase(),Vs=r=>{if(Intl.Segmenter){let t=r?.documentElement?.lang||r?.defaultView?.navigator?.language||void 0;return new Intl.Segmenter(t,{granularity:"word"})}},Xs=(r,t)=>{if(!t.has(r.currentNode)){let o=r.firstChild();if(o!==null)return o}let e=r.nextSibling();if(e!==null)return e;let i=r.parentNode();return i!==null&&t.add(i),i};function _t(r){return r.otherLocations?.get("cssSelector")}function Ki(r){return De.deserialize(r.otherLocations?.get("domRange"))}function Yi(r){return r.fragments?.map(t=>Xi(t)).find(t=>t)}function qi(r){return new Map(r.fragments?.map(t=>t.startsWith("#")?t.slice(1):t).join("&").split("&").filter(t=>!t.startsWith("#")).map(t=>t.split("=")).filter(t=>t.length===2).map(t=>[t[0].trim().toLowerCase(),t[1].trim()]))}function _e(r){if(!r.fragments?.length)return;let t=r.fragments.find(e=>e.length&&!e.includes("="));if(!t){let e=qi(r);e.has("id")?t=e.get("id"):e.has("name")&&(t=e.get("name"))}return t?.startsWith("#")?t.slice(1):t}function Ji(r){let t=qi(r).get("t");if(t)return Mt(t)}var Y=class r{constructor(t){let e,i,o=t.mediaType.replace(/\s/g,"").split(";"),n=o[0].split("/");if(n.length===2){if(e=n[0].toLowerCase().trim(),i=n[1].toLowerCase().trim(),e.length===0||i.length===0)throw new Error("Invalid media type")}else throw new Error("Invalid media type");let s={};for(let m=1;m<o.length;m++){let p=o[m].split("=");if(p.length===2){let g=p[0].toLocaleLowerCase(),u=g==="charset"?p[1].toUpperCase():p[1];s[g]=u}}let a={},h=Object.keys(s);h.sort((m,p)=>m.localeCompare(p)),h.forEach(m=>a[m]=s[m]);let l="";for(let m in a){let p=a[m];l+=`;${m}=${p}`}let d=`${e}/${i}${l}`,c=a.encoding;this.string=d,this.type=e,this.subtype=i,this.parameters=a,this.encoding=c,this.name=t.name,this.fileExtension=t.fileExtension}static parse(t){return new r(t)}get structuredSyntaxSuffix(){let t=this.subtype.split("+");return t.length>1?`+${t[t.length-1]}`:void 0}get essence(){return`${this.type}/${this.subtype}`}get charset(){return this.parameters.charset}contains(t){let e=typeof t=="string"?r.parse({mediaType:t}):t;if(!((this.type==="*"||this.type===e.type)&&(this.subtype==="*"||this.subtype===e.subtype)))return!1;let i=new Set(Object.entries(this.parameters).map(([n,s])=>`${n}=${s}`)),o=new Set(Object.entries(e.parameters).map(([n,s])=>`${n}=${s}`));for(let n of Array.from(i.values()))if(!o.has(n))return!1;return!0}matches(t){let e=typeof t=="string"?r.parse({mediaType:t}):t;return this.contains(e)||e.contains(this)}matchesAny(...t){for(let e of t)if(this.matches(e))return!0;return!1}equals(t){return this.string===t.string}get isZIP(){return this.matchesAny(r.ZIP,r.LCP_PROTECTED_AUDIOBOOK,r.LCP_PROTECTED_PDF)||this.structuredSyntaxSuffix==="+zip"}get isJSON(){return this.matchesAny(r.JSON)||this.structuredSyntaxSuffix==="+json"}get isOPDS(){return this.matchesAny(r.OPDS1,r.OPDS1_ENTRY,r.OPDS2,r.OPDS2_PUBLICATION,r.OPDS_AUTHENTICATION)||this.structuredSyntaxSuffix==="+json"}get isHTML(){return this.matchesAny(r.HTML,r.XHTML)}get isBitmap(){return this.matchesAny(r.AVIF,r.BMP,r.GIF,r.JPEG,r.PNG,r.TIFF,r.WEBP)}get isAudio(){return this.type==="audio"}get isVideo(){return this.type==="video"}get isRWPM(){return this.matchesAny(r.READIUM_AUDIOBOOK_MANIFEST,r.DIVINA_MANIFEST,r.READIUM_WEBPUB_MANIFEST)}get isPublication(){return this.matchesAny(r.READIUM_AUDIOBOOK,r.READIUM_AUDIOBOOK_MANIFEST,r.CBZ,r.DIVINA,r.DIVINA_MANIFEST,r.EPUB,r.LCP_PROTECTED_AUDIOBOOK,r.LCP_PROTECTED_PDF,r.LPF,r.PDF,r.W3C_WPUB_MANIFEST,r.READIUM_WEBPUB,r.READIUM_WEBPUB_MANIFEST,r.ZAB)}static get AAC(){return r.parse({mediaType:"audio/aac",fileExtension:"aac"})}static get ACSM(){return r.parse({mediaType:"application/vnd.adobe.adept+xml",name:"Adobe Content Server Message",fileExtension:"acsm"})}static get AIFF(){return r.parse({mediaType:"audio/aiff",fileExtension:"aiff"})}static get AVI(){return r.parse({mediaType:"video/x-msvideo",fileExtension:"avi"})}static get AVIF(){return r.parse({mediaType:"image/avif",fileExtension:"avif"})}static get BINARY(){return r.parse({mediaType:"application/octet-stream"})}static get BMP(){return r.parse({mediaType:"image/bmp",fileExtension:"bmp"})}static get CBZ(){return r.parse({mediaType:"application/vnd.comicbook+zip",name:"Comic Book Archive",fileExtension:"cbz"})}static get CSS(){return r.parse({mediaType:"text/css",fileExtension:"css"})}static get DIVINA(){return r.parse({mediaType:"application/divina+zip",name:"Digital Visual Narratives",fileExtension:"divina"})}static get DIVINA_MANIFEST(){return r.parse({mediaType:"application/divina+json",name:"Digital Visual Narratives",fileExtension:"json"})}static get EPUB(){return r.parse({mediaType:"application/epub+zip",name:"EPUB",fileExtension:"epub"})}static get GIF(){return r.parse({mediaType:"image/gif",fileExtension:"gif"})}static get GZ(){return r.parse({mediaType:"application/gzip",fileExtension:"gz"})}static get HTML(){return r.parse({mediaType:"text/html",fileExtension:"html"})}static get JAVASCRIPT(){return r.parse({mediaType:"text/javascript",fileExtension:"js"})}static get JPEG(){return r.parse({mediaType:"image/jpeg",fileExtension:"jpeg"})}static get JSON(){return r.parse({mediaType:"application/json"})}static get LCP_LICENSE_DOCUMENT(){return r.parse({mediaType:"application/vnd.readium.lcp.license.v1.0+json",name:"LCP License",fileExtension:"lcpl"})}static get LCP_PROTECTED_AUDIOBOOK(){return r.parse({mediaType:"application/audiobook+lcp",name:"LCP Protected Audiobook",fileExtension:"lcpa"})}static get LCP_PROTECTED_PDF(){return r.parse({mediaType:"application/pdf+lcp",name:"LCP Protected PDF",fileExtension:"lcpdf"})}static get LCP_STATUS_DOCUMENT(){return r.parse({mediaType:"application/vnd.readium.license.status.v1.0+json"})}static get LPF(){return r.parse({mediaType:"application/lpf+zip",fileExtension:"lpf"})}static get MP3(){return r.parse({mediaType:"audio/mpeg",fileExtension:"mp3"})}static get MPEG(){return r.parse({mediaType:"video/mpeg",fileExtension:"mpeg"})}static get NCX(){return r.parse({mediaType:"application/x-dtbncx+xml",fileExtension:"ncx"})}static get OGG(){return r.parse({mediaType:"audio/ogg",fileExtension:"oga"})}static get OGV(){return r.parse({mediaType:"video/ogg",fileExtension:"ogv"})}static get OPDS1(){return r.parse({mediaType:"application/atom+xml;profile=opds-catalog"})}static get OPDS1_ENTRY(){return r.parse({mediaType:"application/atom+xml;type=entry;profile=opds-catalog"})}static get OPDS2(){return r.parse({mediaType:"application/opds+json"})}static get OPDS2_PUBLICATION(){return r.parse({mediaType:"application/opds-publication+json"})}static get OPDS_AUTHENTICATION(){return r.parse({mediaType:"application/opds-authentication+json"})}static get OPUS(){return r.parse({mediaType:"audio/opus",fileExtension:"opus"})}static get OTF(){return r.parse({mediaType:"font/otf",fileExtension:"otf"})}static get PDF(){return r.parse({mediaType:"application/pdf",name:"PDF",fileExtension:"pdf"})}static get PNG(){return r.parse({mediaType:"image/png",fileExtension:"png"})}static get READIUM_AUDIOBOOK(){return r.parse({mediaType:"application/audiobook+zip",name:"Readium Audiobook",fileExtension:"audiobook"})}static get READIUM_AUDIOBOOK_MANIFEST(){return r.parse({mediaType:"application/audiobook+json",name:"Readium Audiobook",fileExtension:"json"})}static get READIUM_CONTENT_DOCUMENT(){return r.parse({mediaType:"application/vnd.readium.content+json",name:"Readium Content Document",fileExtension:"json"})}static get READIUM_GUIDED_NAVIGATION_DOCUMENT(){return r.parse({mediaType:"application/guided-navigation+json",name:"Readium Guided Navigation Document",fileExtension:"json"})}static get READIUM_POSITION_LIST(){return r.parse({mediaType:"application/vnd.readium.position-list+json",name:"Readium Position List",fileExtension:"json"})}static get READIUM_WEBPUB(){return r.parse({mediaType:"application/webpub+zip",name:"Readium Web Publication",fileExtension:"webpub"})}static get READIUM_WEBPUB_MANIFEST(){return r.parse({mediaType:"application/webpub+json",name:"Readium Web Publication",fileExtension:"json"})}static get SMIL(){return r.parse({mediaType:"application/smil+xml",fileExtension:"smil"})}static get SVG(){return r.parse({mediaType:"image/svg+xml",fileExtension:"svg"})}static get TEXT(){return r.parse({mediaType:"text/plain",fileExtension:"txt"})}static get TIFF(){return r.parse({mediaType:"image/tiff",fileExtension:"tiff"})}static get TTF(){return r.parse({mediaType:"font/ttf",fileExtension:"ttf"})}static get W3C_WPUB_MANIFEST(){return r.parse({mediaType:"application/x.readium.w3c.wpub+json",name:"Web Publication",fileExtension:"json"})}static get WAV(){return r.parse({mediaType:"audio/wav",fileExtension:"wav"})}static get WEBM_AUDIO(){return r.parse({mediaType:"audio/webm",fileExtension:"webm"})}static get WEBM_VIDEO(){return r.parse({mediaType:"video/webm",fileExtension:"webm"})}static get WEBP(){return r.parse({mediaType:"image/webp",fileExtension:"webp"})}static get WOFF(){return r.parse({mediaType:"font/woff",fileExtension:"woff"})}static get WOFF2(){return r.parse({mediaType:"font/woff2",fileExtension:"woff2"})}static get XHTML(){return r.parse({mediaType:"application/xhtml+xml",fileExtension:"xhtml"})}static get XML(){return r.parse({mediaType:"application/xml",fileExtension:"xml"})}static get ZAB(){return r.parse({mediaType:"application/x.readium.zab+zip",name:"Zipped Audio Book",fileExtension:"zab"})}static get ZIP(){return r.parse({mediaType:"application/zip",fileExtension:"zip"})}};var I=(r=>(r.left="left",r.right="right",r.center="center",r))(I||{}),te=class r{constructor(t){this.otherProperties=t}get page(){return this.otherProperties.page}static deserialize(t){if(t)return new r(t)}serialize(){return this.otherProperties}add(t){let e=Object.assign({},this.otherProperties);for(let i in t)e[i]=t[i];return new r(e)}};var At=class{constructor(t){this.uri=t,this.parameters=this.getParameters(t)}getParameters(t){let e=/\{\??([^}]+)\}/g,i=t.match(e);return i?new Set(i.join(",").replace(e,"$1").split(",").map(o=>o.trim())):new Set}expand(t){let e=o=>o.split(",").map(n=>{let s=t[n];return s?encodeURIComponent(s):""}).join(","),i=o=>"?"+o.split(",").map(n=>{let s=n.split("=")[0],a=t[s];return a?`${s}=${encodeURIComponent(a)}`:""}).join("&");return this.uri.replace(/\{(\??)([^}]+)\}/g,(...o)=>o[1]?i(o[2]):e(o[2]))}};var G=class r{constructor(t){this.fragments=t.fragments?t.fragments:new Array,this.progression=t.progression,this.totalProgression=t.totalProgression,this.position=t.position,this.otherLocations=t.otherLocations}static deserialize(t){if(!t)return;let e=zt(t.progression),i=zt(t.totalProgression),o=zt(t.position),n=new Map,s=new Set(["fragment","fragments","progression","totalProgression","position","otherLocations"]);return Object.entries(t).forEach(([a,h])=>{s.has(a)||n.set(a,h)}),t.otherLocations?.forEach?.((a,h)=>n.set(h,a)),new r({fragments:A(t.fragments||t.fragment),progression:e!==void 0&&e>=0&&e<=1?e:void 0,totalProgression:i!==void 0&&i>=0&&i<=1?i:void 0,position:o!==void 0&&o>0?o:void 0,otherLocations:n.size===0?void 0:n})}serialize(){let t={};return this.fragments&&(t.fragments=this.fragments),this.progression!==void 0&&(t.progression=this.progression),this.totalProgression!==void 0&&(t.totalProgression=this.totalProgression),this.position!==void 0&&(t.position=this.position),this.otherLocations&&this.otherLocations.forEach((e,i)=>t[i]=e),t}},at=class r{constructor(t){this.after=t.after,this.before=t.before,this.highlight=t.highlight}static deserialize(t){if(t)return new r({after:t.after,before:t.before,highlight:t.highlight})}serialize(){let t={};return this.after!==void 0&&(t.after=this.after),this.before!==void 0&&(t.before=this.before),this.highlight!==void 0&&(t.highlight=this.highlight),t}},z=class r{constructor(t){let e=t.href.indexOf("#"),i=e>=0?t.href.slice(e+1):void 0;this.href=e>=0?t.href.slice(0,e):t.href,this.type=t.type,this.title=t.title;let o=t.locations?.fragments,n=i&&(!o||o.length===0);this.locations=t.locations?n?new G({...t.locations,fragments:[i]}):t.locations:i?new G({fragments:[i]}):new G({}),this.text=t.text}static deserialize(t){if(t&&t.href&&t.type)return new r({href:t.href,type:t.type,title:t.title,locations:G.deserialize(t.locations),text:at.deserialize(t.text)})}serialize(){let t={href:this.href,type:this.type};return this.title!==void 0&&(t.title=this.title),this.locations&&(t.locations=this.locations.serialize()),this.text&&(t.text=this.text.serialize()),t}copyWithLocations(t){return new r({href:this.href,type:this.type,title:this.title,text:this.text,locations:new G({...this.locations,...t})})}};var q=class r{constructor(t){this.href=t.href,this.templated=t.templated,this.type=t.type,this.title=t.title,this.rels=t.rels,this.properties=t.properties,this.height=t.height,this.width=t.width,this.size=t.size,this.duration=t.duration,this.bitrate=t.bitrate,this.languages=t.languages,this.alternates=t.alternates,this.children=t.children}static deserialize(t){if(!(!t||typeof t.href!="string"))return new r({href:t.href,templated:t.templated,type:t.type,title:t.title,rels:t.rel?Array.isArray(t.rel)?new Set(t.rel):new Set([t.rel]):void 0,properties:te.deserialize(t.properties),height:rt(t.height),width:rt(t.width),size:rt(t.size),duration:rt(t.duration),bitrate:rt(t.bitrate),languages:A(t.language),alternates:D.deserialize(t.alternate),children:D.deserialize(t.children)})}serialize(){let t={href:this.href};return this.templated!==void 0&&(t.templated=this.templated),this.type!==void 0&&(t.type=this.type),this.title!==void 0&&(t.title=this.title),this.rels&&(t.rel=Lt(this.rels)),this.properties&&(t.properties=this.properties.serialize()),this.height!==void 0&&(t.height=this.height),this.width!==void 0&&(t.width=this.width),this.size!==void 0&&(t.size=this.size),this.duration!==void 0&&(t.duration=this.duration),this.bitrate!==void 0&&(t.bitrate=this.bitrate),this.languages&&(t.language=this.languages),this.alternates&&(t.alternate=this.alternates.serialize()),this.children&&(t.children=this.children.serialize()),t}get mediaType(){return this.type!==void 0?Y.parse({mediaType:this.type}):Y.BINARY}toURL(t){let e=this.href.replace(/^(\/)/,"");if(e.length===0)return;let i=t||"/";return i.startsWith("/")&&(i="file://"+i),new URL(e,i).href.replace(/^(file:\/\/)/,"")}get templateParameters(){return this.templated?new At(this.href).parameters:new Set}expandTemplate(t){return new r({href:new At(this.href).expand(t),templated:!1})}addProperties(t){let e=r.deserialize(this.serialize());return e.properties=e.properties?e.properties?.add(t):new te(t),e}get locator(){return new z({href:this.href,type:this.type??"",title:this.title})}},D=class r{constructor(t){this.items=t}static deserialize(t){if(t&&Array.isArray(t))return new r(t.map(e=>q.deserialize(e)).filter(e=>e!==void 0))}serialize(){return this.items.map(t=>t.serialize())}findWithRel(t){let e=i=>i.rels&&i.rels.has(t);return this.items.find(e)}filterByRel(t){let e=i=>i.rels&&i.rels.has(t);return this.items.filter(e)}findWithHref(t){let e=i=>i.href===t;return this.items.find(e)}findIndexWithHref(t){let e=i=>i.href===t;return this.items.findIndex(e)}findWithMediaType(t){let e=i=>i.mediaType.matches(t);return this.items.find(e)}filterByMediaType(t){let e=i=>i.mediaType.matches(t);return this.items.filter(e)}filterByMediaTypes(t){let e=i=>{for(let o of t)if(i.mediaType.matches(o))return!0;return!1};return this.items.filter(e)}everyIsAudio(){let t=e=>e.mediaType.isAudio;return this.items.length>0&&this.items.every(t)}everyIsBitmap(){let t=e=>e.mediaType.isBitmap;return this.items.length>0&&this.items.every(t)}everyIsHTML(){let t=e=>e.mediaType.isHTML;return this.items.length>0&&this.items.every(t)}everyIsVideo(){let t=e=>e.mediaType.isVideo;return this.items.length>0&&this.items.every(t)}everyMatchesMediaType(t){return Array.isArray(t)?this.items.length>0&&this.items.every(e=>{for(let i of t)return e.mediaType.matches(i);return!1}):this.items.length>0&&this.items.every(e=>e.mediaType.matches(t))}filterLinksHasType(){return this.items.filter(t=>t.type)}};var Z=(r=>(r.EPUB="https://readium.org/webpub-manifest/profiles/epub",r.AUDIOBOOK="https://readium.org/webpub-manifest/profiles/audiobook",r.DIVINA="https://readium.org/webpub-manifest/profiles/divina",r.PDF="https://readium.org/webpub-manifest/profiles/pdf",r))(Z||{});var ee=class r{constructor(t,e,i=[],o=[],n=void 0){this._trimmedLinkMap=new Map,this._allItems=t,this.linkMap=e,this._conformsTo=i,this.tocLinks=o,this._tocDepth=n}static build(t,e={}){let i=t.toc?.items??[],o=t.readingOrder.items,{depth:n}=e,s=t.metadata?.conformsTo??[],a=new Map,h=[];for(let l=0;l<o.length;l++){let d=o[l],c=r.bareHref(d.href),m=d.title??r.findTitleInToc(i,c,n),p=r.collectChildrenFromToc(i,c,n,1,a),g={title:m,references:[d.href],children:p.length>0?p:void 0};a.set(g,d),h.push(g)}return new r(h,a,s,i,n)}augment(t){for(let e of this.flatAll){let i=this.linkFor(e);if(!i)continue;let o=t(e,i);o.position!==void 0&&(e.position=o.position),o.scroll!==void 0&&(e.scroll=o.scroll),o.role!==void 0&&(e.role=o.role)}}get depth(){return this._depth}set depth(t){this._depth!==t&&(this._depth=t,this._tocDepth=t,this._items=void 0,this._flat=void 0,this._toc=void 0)}get items(){return this._items||(this._depth!==void 0?(this._trimmedLinkMap=new Map,this._items=r.trimToDepth(this._allItems,this._depth,this.linkMap,this._trimmedLinkMap)):this._items=this._allItems),this._items}locate(t){let e=t.href.split("#")[0],i=Ji(t.locations),o=_e(t.locations),n=t.locations.progression;if(i!==void 0){let s=-1/0,a;for(let h of this.flat){if(!this.itemMatchesHref(h,e))continue;let l=this.itemStartTime(h,e);l!==void 0&&l<=i&&l>s&&(s=l,a=h)}if(a)return a}if(o){let s=this.flat.find(a=>this.itemMatchesHref(a,e)?a.references.some(h=>h.split("#")[1]===o):!1);if(s)return s}if(n!==void 0){let s=-1/0,a;for(let l of this.flat){if(!this.itemMatchesHref(l,e))continue;let d=this.itemScrollPosition(l);d!==void 0&&d<=n&&d>s&&(s=d,a=l)}if(a)return a;let h=this.items.find(l=>this.itemMatchesHref(l,e));if(h?.children?.length&&!h.children.some(l=>l.scroll!==void 0)){let l=Math.min(Math.floor(n*h.children.length),h.children.length-1);return h.children[l]}}return this.flat.find(s=>this.itemMatchesHref(s,e))}navigableFrom(t){let e=this.flat.indexOf(t);return{previous:e>0?this.flat[e-1]:void 0,next:e>=0&&e<this.flat.length-1?this.flat[e+1]:void 0}}segmentsForHref(t){let e=t.split("#")[0],i=this.items.find(o=>this.itemMatchesHref(o,e));return i?i.children?.length?i.children:[i]:[]}ancestors(t){return this.ancestorPath(this.items,t)??[]}linkFor(t){return this.linkMap.get(t)??this._trimmedLinkMap.get(t)}get contextualizedToc(){return this._toc||(this._toc=this.tocLinks.length>0?this.buildTocEntries(this.tocLinks,this._tocDepth,1):this._allItems.map(t=>this.entryFor(this.linkFor(t),t))),this._toc}tocEntryFor(t){let e=this.linkFor(t);if(!e)return;let i=this.findTocEntryByLink(this.contextualizedToc,e);return i||this.nearestTocEntryForResource(e.href,t)||this.previousResolvedTocEntry(e)}get flat(){return this._flat||(this._flat=this.flattenItems(this.items)),this._flat}get flatAll(){return this.flattenItems(this._allItems)}static findTitleInToc(t,e,i){let{atStart:o,fragments:n}=r.collectTocCandidates(t,e,i,1);if(o.length>0)return o[0].title;if(n.length===1)return n[0].title}static collectChildrenFromToc(t,e,i,o,n){if(i!==void 0&&o>i)return[];let s=[];for(let a of t){let h=r.bareHref(a.href),l=a.href.startsWith("#");if((h===e||l&&h==="")&&a.title&&!r.isStartOfResource(a.href)){let d=h===""?e+a.href:a.href,c={title:a.title,references:[d]};n.set(c,a),s.push(c)}a.children?.items?.length&&s.push(...r.collectChildrenFromToc(a.children.items,e,i,o+1,n))}return s}static collectTocCandidates(t,e,i,o){if(i!==void 0&&o>i)return{atStart:[],fragments:[]};let n=[],s=[];for(let a of t){let h=r.bareHref(a.href),l=a.href.startsWith("#");if((h===e||l&&h==="")&&a.title&&(r.isStartOfResource(a.href)?n.push(a):s.push(a)),a.children?.items?.length){let d=r.collectTocCandidates(a.children.items,e,i,o+1);n.push(...d.atStart),s.push(...d.fragments)}}return{atStart:n,fragments:s}}static isStartOfResource(t){let e=t.split("#")[1];if(!e)return!0;let i=e.match(/(?:^|&)t=([^&]+)/);return i!==null&&Xo(i[1])}get linkToItem(){if(!this._linkToItem){this._linkToItem=new Map;for(let[t,e]of this.linkMap)this._linkToItem.set(e,t)}return this._linkToItem}buildTocEntries(t,e,i){return e!==void 0&&i>e?[]:t.map(o=>{let n=this.linkToItem.get(o)??this.resourceStartItem(o.href),s=o.children?.items?.length?this.buildTocEntries(o.children.items,e,i+1):[];return{...this.entryFor(o,n),children:s.length>0?s:void 0}})}tocLinksFlat(t,e,i){if(e!==void 0&&i>e)return[];let o=[];for(let n of t)o.push(n),n.children?.items?.length&&o.push(...this.tocLinksFlat(n.children.items,e,i+1));return o}entryFor(t,e){let i=this._conformsTo.includes(Z.AUDIOBOOK);return{link:t,position:!i&&e?.position!==void 0?String(e.position):void 0,timestamp:i&&e?.position!==void 0?$o(e.position):void 0}}resourceStartItem(t){let e=r.bareHref(t);return this._allItems.find(i=>this.itemMatchesHref(i,e))}findTocEntryByLink(t,e){for(let i of t){if(i.link===e)return i;if(i.children){let o=this.findTocEntryByLink(i.children,e);if(o)return o}}}nearestTocEntryForResource(t,e){let i=r.bareHref(t),o=this.tocLinksFlat(this.tocLinks,this._tocDepth,1).filter(d=>r.bareHref(d.href)===i).map(d=>({link:d,item:this.linkToItem.get(d)??this.resourceStartItem(d.href)}));if(o.length===0)return;let n=this._conformsTo.includes(Z.AUDIOBOOK),s=n?e.position??0:e.scroll??0,a,h=-1/0;for(let d of o){let c=n?d.item?.position??0:d.item?.scroll??0;c<=s&&c>h&&(h=c,a=d)}let l=a??o[0];return this.findTocEntryByLink(this.contextualizedToc,l.link)}previousResolvedTocEntry(t){let e=this._allItems.findIndex(i=>this.linkFor(i)===t);if(e!==-1)for(let i=e-1;i>=0;i--){let o=this.linkFor(this._allItems[i]);if(!o)continue;let n=r.bareHref(o.href),{atStart:s,fragments:a}=r.collectTocCandidates(this.tocLinks,n,this._tocDepth,1),h=s.filter(m=>r.bareHref(m.href)===n),l=a.filter(m=>r.bareHref(m.href)===n),d=h[0]??(l.length>0?l[l.length-1]:void 0);if(!d)continue;let c=this.findTocEntryByLink(this.contextualizedToc,d);if(c)return c}}static trimToDepth(t,e,i,o){return t.map(n=>{let s={...n,children:e>1&&n.children?.length?r.trimToDepth(n.children,e-1,i,o):void 0},a=i.get(n);return a&&o.set(s,a),s})}flattenItems(t){let e=[];for(let i of t)e.push(i),i.children&&e.push(...this.flattenItems(i.children));return e}itemStartTime(t,e){for(let i of t.references){let o=i.indexOf("#"),n=o>=0?i.slice(0,o):i,s=o>=0?i.slice(o+1):void 0;if((n||e)!==e)continue;if(!s)return;let a=s.match(/(?:^|&)t=([^&]+)/);return a?Mt(a[1]):void 0}}itemMatchesHref(t,e){return t.references.some(i=>{let o=i.split("#")[0];return o===e||o===""})}ancestorPath(t,e){for(let i of t){if(i===e)return[];if(i.children){let o=this.ancestorPath(i.children,e);if(o!==null)return[i,...o]}}return null}itemScrollPosition(t){return t.scroll}static bareHref(t){return t.split("#")[0]}};function en(r,t){return ee.build(r,t)}function on(r,t){let e=ee.build(r,t),i=r.readingOrder.items;if(!(i.length<=1||i.every(s=>s.duration!==void 0)))return e;let o=0,n=new Map(i.map(s=>{let a=o;return o+=s.duration??0,[s.href,a]}));return e.augment((s,a)=>{let h=a.href,l=h.indexOf("#"),d=l>=0?h.slice(0,l):h,c=l>=0?h.slice(l+1):void 0,m=n.get(d)??0,p=c?.match(/(?:^|&)t=([^&]+)/),g=p?Mt(p[1]):void 0;return{position:m+(g??0)}}),e}function nn(r,t){return r.metadata?.conformsTo?.includes(Z.AUDIOBOOK)??!1?on(r,t):en(r,t)}var Dt=class r{constructor(t){this.value=t.value,this.scheme=t.scheme}static deserialize(t){if(t){if(typeof t=="string")return new r({value:t});if(typeof t=="object"&&t.value)return new r({value:t.value,scheme:t.scheme})}}serialize(){return this.scheme?{value:this.value,scheme:this.scheme}:this.value}};var Zi=class ie{constructor(t){this.translations=typeof t=="string"?{[ie.UNDEFINED_LANGUAGE]:t}:t}static deserialize(t){if(t&&(typeof t=="string"||t.constructor===Object))return new ie(t)}serialize(){return this.translations}getTranslation(t){return this.translations[t||ie.UNDEFINED_LANGUAGE]||this.translations[ie.UNDEFINED_LANGUAGE]||this.translations[ie.LANGUAGE_EN]||(Object.values(this.translations).length===0?"":Object.values(this.translations)[0])}};Zi.UNDEFINED_LANGUAGE="und",Zi.LANGUAGE_EN="en";var ht=Zi;var Qi=class r{constructor(t){this.name=t.name,this.sortAs=t.sortAs,this.identifier=t.identifier,this.altIdentifiers=t.altIdentifiers,this.roles=t.roles,this.links=t.links,this.position=t.position}static deserialize(t){if(t)return typeof t=="string"?new r({name:ht.deserialize(t)}):t.name?new r({name:ht.deserialize(t.name),sortAs:ht.deserialize(t.sortAs),identifier:t.identifier,altIdentifiers:t.altIdentifier?Array.isArray(t.altIdentifier)?new Set(t.altIdentifier.map(e=>Dt.deserialize(e)).filter(e=>e!==void 0)):new Set([Dt.deserialize(t.altIdentifier)].filter(e=>e!==void 0)):void 0,roles:t.role?new Set(A(t.role)):void 0,links:D.deserialize(t.links),position:zt(t.position)}):void 0}serialize(){let t={name:this.name.serialize()};return this.sortAs!==void 0&&(t.sortAs=this.sortAs.serialize()),this.identifier!==void 0&&(t.identifier=this.identifier),this.altIdentifiers&&(t.altIdentifier=Lt(this.altIdentifiers).map(e=>e.serialize())),this.roles&&(t.role=Lt(this.roles)),this.links&&(t.links=this.links.serialize()),this.position!==void 0&&(t.position=this.position),t}},U=class r{constructor(t){this.items=t}static deserialize(t){if(!t)return;let e=Array.isArray(t)?t:[t];return new r(e.map(i=>Qi.deserialize(i)).filter(i=>i!==void 0))}serialize(){return this.items.map(t=>t.serialize())}};var oe=class r{constructor(t){this.items=t&&t.items?t.items:new Map}static deserialize(t){if(!(t&&typeof t=="object"))return;let e=new Map;return Object.entries(t).forEach(([i,o])=>{let n=U.deserialize(o);n&&n.items.length>0&&e.set(i,n)}),new r({items:e})}serialize(){let t={};return this.items.forEach((e,i)=>t[i]=e.serialize()),t}};var E=(r=>(r.ltr="ltr",r.rtl="rtl",r))(E||{});var to=class r{constructor(t){this.name=t.name,this.sortAs=t.sortAs,this.code=t.code,this.scheme=t.scheme,this.links=t.links}static deserialize(t){if(t)return typeof t=="string"?new r({name:ht.deserialize(t)}):t.name?new r({name:ht.deserialize(t.name),sortAs:t.sortAs?ht.deserialize(t.sortAs):void 0,code:t.code,scheme:t.scheme,links:D.deserialize(t.links)}):void 0}serialize(){let t={name:this.name.serialize()};return this.sortAs!==void 0&&(t.sortAs=this.sortAs.serialize()),this.code!==void 0&&(t.code=this.code),this.scheme!==void 0&&(t.scheme=this.scheme),this.links&&(t.links=this.links.serialize()),t}},Xe=class r{constructor(t){this.items=t}static deserialize(t){if(!t)return;let e=Array.isArray(t)?t:[t];return new r(e.map(i=>to.deserialize(i)).filter(i=>i!==void 0))}serialize(){return this.items.map(t=>t.serialize())}};var $s=(r=>(r.all="all",r.none="none",r))($s||{}),$e=class r{constructor(t){this.reservation=t.reservation,this.policy=t.policy}static deserialize(t){if(t)return new r({reservation:t.reservation,policy:t.policy})}serialize(){let t={};return this.reservation!==void 0&&(t.reservation=this.reservation),this.policy!==void 0&&(t.policy=this.policy),t}};var rn=class eo{constructor(t){this.title=t.title,this.typeUri=t.typeUri,this.conformsTo=t.conformsTo,this.identifier=t.identifier,this.altIdentifier=t.altIdentifier,this.subtitle=t.subtitle,this.sortAs=t.sortAs,this.artists=t.artists,this.authors=t.authors,this.colorists=t.colorists,this.contributors=t.contributors,this.editors=t.editors,this.illustrators=t.illustrators,this.inkers=t.inkers,this.letterers=t.letterers,this.narrators=t.narrators,this.pencilers=t.pencilers,this.translators=t.translators,this.languages=t.languages,this.description=t.description,this.publishers=t.publishers,this.imprints=t.imprints,this.published=t.published,this.modified=t.modified,this.subjects=t.subjects,this.belongsTo=t.belongsTo,this.belongsToCollections=t.belongsToCollections,this.belongsToSeries=t.belongsToSeries,this.belongsToCollections&&this.belongsToCollections.items.length>0&&(this.belongsTo||(this.belongsTo=new oe),this.belongsTo.items.set("collection",this.belongsToCollections)),this.belongsToSeries&&this.belongsToSeries.items.length>0&&(this.belongsTo||(this.belongsTo=new oe),this.belongsTo.items.set("series",this.belongsToSeries)),this.layout=t.layout,this.readingProgression=t.readingProgression,this.duration=t.duration,this.numberOfPages=t.numberOfPages,this.accessibility=t.accessibility,this.tdm=t.tdm,this.otherMetadata=t.otherMetadata}static deserialize(t){if(!(t&&t.title))return;let e=ht.deserialize(t.title),i=t["@type"],o=A(t.conformsTo),n=t.identifier,s=Dt.deserialize(t.altIdentifier),a=ht.deserialize(t.subtitle),h=ht.deserialize(t.sortAs),l=U.deserialize(t.artist),d=U.deserialize(t.author),c=U.deserialize(t.colorist),m=U.deserialize(t.contributor),p=U.deserialize(t.editor),g=U.deserialize(t.illustrator),u=U.deserialize(t.inker),f=U.deserialize(t.letterer),S=U.deserialize(t.narrator),b=U.deserialize(t.penciler),C=U.deserialize(t.translator),v=A(t.language),w=t.description,B=U.deserialize(t.publisher),it=U.deserialize(t.imprint),lt=Wi(t.published),Di=Wi(t.modified),Ui=Xe.deserialize(t.subject),Ii=oe.deserialize(t.belongsTo),fs=Ae.deserialize(t.accessibility),Ss=t.layout,ys=t.readingProgression,_s=rt(t.duration),bs=rt(t.numberOfPages),vs=$e.deserialize(t.tdm),Le=Object.assign({},t);return eo.mappedProperties.forEach(Rs=>delete Le[Rs]),Object.keys(Le).length===0&&(Le=void 0),new eo({title:e,typeUri:i,conformsTo:o,identifier:n,altIdentifier:s,subtitle:a,sortAs:h,artists:l,authors:d,colorists:c,contributors:m,editors:p,illustrators:g,inkers:u,letterers:f,narrators:S,pencilers:b,translators:C,languages:v,description:w,publishers:B,imprints:it,published:lt,modified:Di,subjects:Ui,belongsTo:Ii,layout:Ss,readingProgression:ys,duration:_s,numberOfPages:bs,accessibility:fs,tdm:vs,otherMetadata:Le})}serialize(){let t={title:this.title.serialize()};if(this.typeUri!==void 0&&(t["@type"]=this.typeUri),this.conformsTo&&(t.conformsTo=this.conformsTo),this.identifier!==void 0&&(t.identifier=this.identifier),this.altIdentifier&&(t.altIdentifier=this.altIdentifier.serialize()),this.subtitle&&(t.subtitle=this.subtitle.serialize()),this.sortAs&&(t.sortAs=this.sortAs.serialize()),this.editors&&(t.editor=this.editors.serialize()),this.artists&&(t.artist=this.artists.serialize()),this.authors&&(t.author=this.authors.serialize()),this.colorists&&(t.colorist=this.colorists.serialize()),this.contributors&&(t.contributor=this.contributors.serialize()),this.illustrators&&(t.illustrator=this.illustrators.serialize()),this.letterers&&(t.letterer=this.letterers.serialize()),this.narrators&&(t.narrator=this.narrators.serialize()),this.pencilers&&(t.penciler=this.pencilers.serialize()),this.translators&&(t.translator=this.translators.serialize()),this.inkers&&(t.inker=this.inkers.serialize()),this.languages&&(t.language=this.languages),this.description!==void 0&&(t.description=this.description),this.publishers&&(t.publisher=this.publishers.serialize()),this.imprints&&(t.imprint=this.imprints.serialize()),this.published!==void 0&&(t.published=this.published.toISOString()),this.modified!==void 0&&(t.modified=this.modified.toISOString()),this.subjects&&(t.subject=this.subjects.serialize()),this.belongsTo&&(t.belongsTo=this.belongsTo.serialize()),this.layout&&(t.layout=this.layout),this.readingProgression&&(t.readingProgression=this.readingProgression),this.duration!==void 0&&(t.duration=this.duration),this.numberOfPages!==void 0&&(t.numberOfPages=this.numberOfPages),this.accessibility&&(t.accessibility=this.accessibility.serialize()),this.tdm&&(t.tdm=this.tdm.serialize()),this.otherMetadata){let e=this.otherMetadata;Object.keys(e).forEach(i=>t[i]=e[i])}return t}get effectiveLayout(){if(!this.conformsTo)return null;for(let t of this.conformsTo)switch(t){case Z.EPUB:return this.layout||_.reflowable;case Z.DIVINA:return this.layout===_.reflowable?_.fixed:this.layout||_.fixed;case Z.AUDIOBOOK:case Z.PDF:return null}return null}get effectiveReadingProgression(){if(this.readingProgression)return this.readingProgression;if(this.languages?.length!==1)return E.ltr;let t=this.languages[0].toLowerCase();if(t==="zh-hant"||t==="zh-tw")return E.rtl;switch(t.split("-")[0]){case"ar":return E.rtl;case"fa":return E.rtl;case"he":return E.rtl;default:return E.ltr}}};rn.mappedProperties=["title","@type","conformsTo","identifier","altIdentifier","subtitle","sortAs","artist","author","colorist","contributor","editor","illustrator","inker","letterer","narrator","penciler","translator","language","description","publisher","imprint","published","modified","subject","belongsTo","layout","readingProgression","duration","numberOfPages","accessibility","tdm"];var sn=rn;var be=class r{constructor(t){this.metadata=t.metadata,this.links=t.links,this.subcollections=t.subcollections}static deserialize(t){if(!t)return;let e,i,o;if(Array.isArray(t))e=D.deserialize(t);else if(typeof t=="object")e=D.deserialize(t.links),i=new Map,t.metadata&&Object.entries(t.metadata).forEach(([n,s])=>{i?.set(n,s)}),o=r.deserializeCollections(t);else return;if(!(!e||e.items.length===0))return new r({metadata:i?.size?i:void 0,links:e,subcollections:o?.size?o:void 0})}static deserializeCollections(t){if(!t)return;let e=new Map;return Object.entries(t).forEach(([i,o])=>{if(i!=="links"&&i!=="metadata"){let n=r.deserialize(o);if(n){let s=new Array;s.push(n),e.set(i,s)}else if(Array.isArray(o)){let s=o.map(a=>r.deserialize(a)).filter(a=>a!==void 0);e.set(i,s)}}}),e.size?e:void 0}serialize(){let t={};return this.metadata&&(t.metadata={},this.metadata.forEach((e,i)=>t.metadata[i]=e)),this.links.items.length&&(t.links=this.links.serialize()),r.serializeCollection(t,this.subcollections),t}static serializeCollection(t,e){e&&e.size>0&&e.forEach((i,o)=>{i.length===1?t[o]=i[0].serialize():t[o]=i.map(n=>n.serialize())})}};var Ke=class r{constructor(t){this.context=t.context,this.metadata=t.metadata,this.links=t.links||new D([]),this.readingOrder=t.readingOrder,this.resources=t.resources,this.toc=t.toc,this.subcollections=t.subcollections}static deserialize(t){if(!t)return;let e=sn.deserialize(t.metadata);if(!e)return;let i=D.deserialize(t.readingOrder?t.readingOrder:t.spine);if(!i)return;let o=new Set(["@context","metadata","links","readingOrder","spine","resources","toc"]),n={};return Object.keys(t).forEach(s=>{o.has(s)||(n[s]=t[s])}),new r({context:A(t["@context"]),metadata:e,links:D.deserialize(t.links),readingOrder:i,resources:D.deserialize(t.resources),toc:D.deserialize(t.toc),subcollections:be.deserializeCollections(n)})}serialize(){let t={};return this.context!==void 0&&(t["@context"]=this.context),t.metadata=this.metadata.serialize(),this.links&&this.links.items.length>0&&(t.links=this.links.serialize()),t.readingOrder=this.readingOrder.serialize(),this.resources&&(t.resources=this.resources.serialize()),this.toc&&(t.toc=this.toc.serialize()),be.serializeCollection(t,this.subcollections),t}linkWithRel(t){let e=new Array;e.push(this.readingOrder),this.resources&&e.push(this.resources),this.links&&e.push(this.links);let i;for(let o of e)if(i=o.findWithRel(t),i!==void 0)return i;return i}linksWithRel(t){let e=[];return e.push(this.readingOrder.filterByRel(t)),this.resources&&e.push(this.resources.filterByRel(t)),this.links&&e.push(this.links.filterByRel(t)),e.reduce((i,o)=>i.concat(o),[])}locatorFromLink(t){let e=t.href.split("#")[0],i=this.linkWithHref(e);if(!i)return;let o=i.type;if(!o)return;let n=t.href.includes("#");return new z({href:t.href,type:o,title:i.title??t.title,locations:new G({progression:n?void 0:0})})}linkWithHref(t){let e=a=>{let h;for(let d of a)if(h=d.findWithHref(t),h!==void 0)return h;let l=new Array;return a.forEach(d=>{let c=[];for(let m of d.items)m.alternates&&c.push(m.alternates),m.children&&c.push(m.children);l.push(...c)}),l.length>0&&(h=e(l)),h},i=[];i.push(this.readingOrder),this.resources&&i.push(this.resources),this.links&&i.push(this.links);let o=e(i);if(o!==void 0)return o;let n=t.split(/[#]/);if(n.length<2)return;let s=n[0];return this.linkWithHref(s)}get baseURL(){let t=this.links?.items.find(i=>i.rels&&i.rels.has("self"));if(!t)return;let e=t.href;if(e)try{let i=new URL(e);return i.search="",i.hash="",i.pathname=i.pathname.substring(0,i.pathname.lastIndexOf("/")+1),i.href}catch{return e.split(/[?#]/)[0].replace(/[^/]*$/,"")}}setSelfLink(t){this.links||(this.links=new D([])),this.links.items=this.links.items.filter(e=>e.rels===void 0||!e.rels?.has("self")),this.links.items.push(new q({href:t,type:Y.READIUM_WEBPUB_MANIFEST.string,rels:new Set(["self"])}))}};var Ye=class r{constructor(t){this.guided=t.guided}static deserialize(t){if(!t)return;let e=oo.deserializeArray(t.guided);if(!(!e||e.length===0))return new r({guided:e})}serialize(){return{guided:this.guided.map(t=>t.serialize())}}},ve=class r{constructor(t){this.plain=t.plain,this.ssml=t.ssml,this.language=t.language}static deserialize(t){if(t!=null){if(typeof t=="string")return t.length>0?new r({plain:t}):void 0;if(t.plain||t.ssml)return new r({plain:t.plain,ssml:t.ssml,language:t.language})}}serialize(){let t={};return this.plain!==void 0&&(t.plain=this.plain),this.ssml!==void 0&&(t.ssml=this.ssml),this.language!==void 0&&(t.language=this.language),Object.keys(t).length>0?t:void 0}},io=class r{constructor(t){this.audioref=t.audioref,this.imgref=t.imgref,this.textref=t.textref,this.videoref=t.videoref,this.text=t.text}static deserialize(t){if(!t)return;if(typeof t=="string"){let i=ve.deserialize(t);return i?new r({text:i}):void 0}let e=new r({audioref:t.audioref,imgref:t.imgref,textref:t.textref,videoref:t.videoref,text:ve.deserialize(t.text)});if(!(e.audioref===void 0&&e.imgref===void 0&&e.textref===void 0&&e.videoref===void 0&&e.text===void 0))return e}serialize(){let t={};this.audioref!==void 0&&(t.audioref=this.audioref),this.imgref!==void 0&&(t.imgref=this.imgref),this.textref!==void 0&&(t.textref=this.textref),this.videoref!==void 0&&(t.videoref=this.videoref);let e=this.text?.serialize();return e!==void 0&&(t.text=e),t}},oo=class r{constructor(t){this.id=t.id,this.audioref=t.audioref,this.children=t.children,this.imgref=t.imgref,this.role=t.role,this.text=t.text,this.textref=t.textref,this.videoref=t.videoref,this.description=t.description}get plainText(){return this.text?.plain}get ssmlText(){return this.text?.ssml}get textLanguage(){return this.text?.language}static deserialize(t){if(!t)return;let e=r.deserializeArray(t.children),i=new r({id:typeof t.id=="string"?t.id:void 0,audioref:t.audioref,children:e&&e.length>0?e:void 0,imgref:t.imgref,role:t.role?new Set(A(t.role)):void 0,text:ve.deserialize(t.text),textref:t.textref,videoref:t.videoref,description:io.deserialize(t.description)});if(!(i.audioref===void 0&&i.imgref===void 0&&i.textref===void 0&&i.videoref===void 0&&i.text===void 0&&i.children===void 0))return i}static deserializeArray(t){if(Array.isArray(t))return t.map(e=>r.deserialize(e)).filter(e=>e!==void 0)}serialize(){let t={};if(this.id!==void 0&&(t.id=this.id),this.audioref!==void 0&&(t.audioref=this.audioref),this.children!==void 0&&(t.children=this.children.map(e=>e.serialize())),this.imgref!==void 0&&(t.imgref=this.imgref),this.role!==void 0&&(t.role=Lt(this.role)),this.text!==void 0){let e=this.text.serialize();e!==void 0&&(t.text=e)}return this.textref!==void 0&&(t.textref=this.textref),this.videoref!==void 0&&(t.videoref=this.videoref),this.description!==void 0&&(t.description=this.description.serialize()),t}get audioFile(){return this.audioref?.split("#")[0]}get audioTime(){if(this.audioref?.includes("#"))return this.audioref.split("#",2)[1]}get textFile(){return this.textref?.split("#")[0]}get fragmentId(){if(this.textref?.includes("#"))return this.textref.split("#",2)[1]}get clip(){let t=this.audioFile;if(!t)return;let e=this.audioTime,i={audioResource:t,fragmentId:this.fragmentId};if(!e)return i;let o=this.parseTimer(e);return i.start=o[0],i.end=o[1],i}parseTimer(t){if(!t||!t.startsWith("t="))return[void 0,void 0];let e=t.substring(2).split(",").map(i=>parseFloat(i));return e.length===1?[isNaN(e[0])?void 0:e[0],void 0]:e.length>2?[void 0,void 0]:[isNaN(e[0])?void 0:e[0],isNaN(e[1])?void 0:e[1]]}};var qe=class{constructor(t){this.fetcher=new Oe,this.guides=new Map,t.fetcher&&(this.fetcher=t.fetcher),this.manifest=t.manifest,this.context=t.manifest.context,this.metadata=t.manifest.metadata,this.links=t.manifest.links,this.readingOrder=t.manifest.readingOrder,this.resources=t.manifest.resources,this.toc=t.manifest.toc,this.subcollections=t.manifest.subcollections}get timeline(){return this._timeline||(this._timeline=nn(this)),this._timeline}get baseURL(){return this.manifest.baseURL}linkWithHref(t){return this.manifest.linkWithHref(t)}linksWithRole(t){let e=this.subcollections?.get(t);return e&&e.length>0?e[0].links:void 0}linksWithRel(t){return this.manifest.linksWithRel(t)}linkWithRel(t){return this.manifest.linkWithRel(t)}getCover(){let t=[this.links,this.resources,this.readingOrder].filter(Boolean);for(let e of t){let i=e.items.find(o=>o.rels?.has("cover"));if(i)return i}for(let e of t){let i=e.items.find(o=>o.mediaType.isBitmap||o.mediaType.matches(Y.SVG));if(i)return i}}async positionsFromManifest(){let t=this.manifest.links?.findWithMediaType("application/vnd.readium.position-list+json");if(t===void 0)return[];let e=await this.get(t).readAsJSON();return e.total?e.positions.map(i=>z.deserialize(i)).filter(i=>i!==void 0):[]}async guideForLink(t,e){let i=a=>a.alternates?.findWithMediaType("application/guided-navigation+json"),o=i(t);if(!o){let a=this.linkWithHref(t.href);a!==void 0&&(o=i(a))}if(o){let a=await this.fetchGuide(o,t);if(!e?.skipAudio||!a||!no(a.guided))return a}let n=this.manifest.links?.findWithMediaType("application/guided-navigation+json");if(!n)return;let s=await this.fetchGuide(n,t);if(!(e?.skipAudio&&s&&no(s.guided)))return s}fetchGuide(t,e){let i=t.href;if(t.templated){let n=new At(i),s={};return n.parameters.has("ref")&&(s.ref=e.href.split("#")[0]),this.loadGuide(n.expand(s))}let o=this.guides.get(i);return o||(o=this.loadGuide(i),this.guides.set(i,o),o.catch(()=>this.guides.delete(i))),o}async loadGuide(t){let e=await this.get(new q({href:t})).readAsJSON();return Ye.deserialize(e)}get(t){return this.fetcher.get(t)}};function no(r){return!!r?.some(t=>t.audioref!==void 0||no(t.children))}var Ks=Math.pow(2,32),an=()=>Math.round(Math.random()*Ks).toString(36),ne=()=>`${Math.round(performance.now())}-${an()}-${an()}`;var vt=1,Re=class{constructor(t){this.destination=null,this.registrar=new Map,this.origin="",this.channelId="",this.receiver=this.receive.bind(this),this.preLog=[],this.wnd=t,t.addEventListener("message",this.receiver)}receive(t){if(t.source===null)throw Error("Event source is null");if(typeof t.data!="object")return;let e=t.data;if(!(!("_readium"in e)||!e._readium||e._readium<=0)){if(e.key==="_ping"){if(!this.destination){if(this.destination=t.source,this.origin=t.origin,this.channelId=e._channel,e._readium!==vt){e._readium>vt?this.send("error",`received comms version ${e._readium} higher than ${vt}`):this.send("error",`received comms version ${e._readium} lower than ${vt}`),this.destination=null,this.origin="",this.channelId="";return}this.send("_pong",void 0),this.preLog.forEach(i=>this.send("log",i)),this.preLog=[]}return}else if(this.channelId){if(e._channel!==this.channelId||t.origin!==this.origin)return}else return;this.handle(e)}}handle(t){let e=this.registrar.get(t.key);if(!e||e.length===0){t.strict&&this.send("_unhandled",t);return}e.forEach(i=>i.cb(t.data,o=>{this.send("_ack",o,t.id)}))}register(t,e,i){Array.isArray(t)||(t=[t]),t.forEach(o=>{let n=this.registrar.get(o);if(n&&n.length>=0){if(n.find(s=>s.module===e))throw new Error(`Trying to register another callback for combination of event ${o} and module ${e}`);n.push({cb:i,module:e}),this.registrar.set(o,n)}else this.registrar.set(o,[{cb:i,module:e}])})}unregister(t,e){Array.isArray(t)||(t=[t]),t.forEach(i=>{let o=this.registrar.get(i);!o||o.length===0||o.splice(o.findIndex(n=>n.module===e),1)})}unregisterAll(t){this.registrar.forEach((e,i)=>this.registrar.set(i,e.filter(o=>o.module!==t)))}log(...t){this.destination?this.send("log",t):this.preLog.push(t)}get ready(){return!!this.destination}destroy(){this.destination=null,this.channelId="",this.preLog=[],this.registrar.clear(),this.wnd.removeEventListener("message",this.receiver)}send(t,e,i=void 0,o=[]){if(!this.destination)throw Error("Attempted to send comms message before destination has been initialized");let n={_readium:vt,_channel:this.channelId,id:i??ne(),key:t,data:e};try{this.destination.postMessage(n,{targetOrigin:this.origin,transfer:o})}catch(s){if(o.length>0)throw s;this.destination.postMessage(n,this.origin,o)}}};function ln(r){return r.split("").reverse().join("")}function Ys(r,t,e){let i=ln(t);return e.map(o=>{let n=Math.max(0,o.end-t.length-o.errors),s=ln(r.slice(n,o.end));return{start:cn(s,i,o.errors).reduce((a,h)=>o.end-h.end<a?o.end-h.end:a,o.end),end:o.end,errors:o.errors}})}function ro(r){return(r|-r)>>31&1}function hn(r,t,e,i){let o=r.P[e],n=r.M[e],s=i>>>31,a=t[e]|s,h=a|n,l=(a&o)+o^o|a,d=n|~(l|o),c=o&l,m=ro(d&r.lastRowMask[e])-ro(c&r.lastRowMask[e]);return d<<=1,c<<=1,c|=s,d|=ro(i)-s,o=c|~(h|d),n=d&h,r.P[e]=o,r.M[e]=n,m}function cn(r,t,e){if(t.length===0)return[];e=Math.min(e,t.length);let i=[],o=32,n=Math.ceil(t.length/o)-1,s={P:new Uint32Array(n+1),M:new Uint32Array(n+1),lastRowMask:new Uint32Array(n+1)};s.lastRowMask.fill(1<<31),s.lastRowMask[n]=1<<(t.length-1)%o;let a=new Uint32Array(n+1),h=new Map,l=[];for(let m=0;m<256;m++)l.push(a);for(let m=0;m<t.length;m+=1){let p=t.charCodeAt(m);if(h.has(p))continue;let g=new Uint32Array(n+1);h.set(p,g),p<l.length&&(l[p]=g);for(let u=0;u<=n;u+=1){g[u]=0;for(let f=0;f<o;f+=1){let S=u*o+f;S>=t.length||t.charCodeAt(S)===p&&(g[u]|=1<<f)}}}let d=Math.max(0,Math.ceil(e/o)-1),c=new Uint32Array(n+1);for(let m=0;m<=d;m+=1)c[m]=(m+1)*o;c[n]=t.length;for(let m=0;m<=d;m+=1)s.P[m]=-1,s.M[m]=0;for(let m=0;m<r.length;m+=1){let p=r.charCodeAt(m),g;p<l.length?g=l[p]:(g=h.get(p),typeof g>"u"&&(g=a));let u=0;for(let f=0;f<=d;f+=1)u=hn(s,g,f,u),c[f]+=u;if(c[d]-u<=e&&d<n&&(g[d+1]&1||u<0)){d+=1,s.P[d]=-1,s.M[d]=0;let f;if(d===n){let S=t.length%o;f=S===0?o:S}else f=o;c[d]=c[d-1]+f-u+hn(s,g,d,u)}else for(;d>0&&c[d]>=e+o;)d-=1;d===n&&c[d]<=e&&(c[d]<e&&i.splice(0,i.length),i.push({start:-1,end:m+1,errors:c[d]}),e=c[d])}return i}function dn(r,t,e){let i=cn(r,t,e);return Ys(r,t,i)}function un(r,t,e){let i=0,o=[];for(;i!==-1;)i=r.indexOf(t,i),i!==-1&&(o.push({start:i,end:i+t.length,errors:0}),i+=1);return o.length>0?o:dn(r,t,e)}function mn(r,t){return t.length===0||r.length===0?0:1-un(r,t,t.length)[0].errors/t.length}function pn(r,t,e={}){if(t.length===0)return null;let i=Math.min(256,t.length/2),o=un(r,t,i);if(o.length===0)return null;let n=a=>{let h=1-a.errors/t.length,l=e.prefix?mn(r.slice(Math.max(0,a.start-e.prefix.length),a.start),e.prefix):1,d=e.suffix?mn(r.slice(a.end,a.end+e.suffix.length),e.suffix):1,c=1;return typeof e.hint=="number"&&(c=1-Math.abs(a.start-e.hint)/r.length),(50*h+20*l+20*d+2*c)/92},s=o.map(a=>({start:a.start,end:a.end,score:n(a)}));return s.sort((a,h)=>h.score-a.score),s[0]}function so(r,t,e){let i=e===1?t:t-1;if(r.charAt(i).trim()!=="")return t;let o,n;if(e===2?(o=r.substring(0,t),n=o.trimEnd()):(o=r.substring(t),n=o.trimStart()),!n.length)return-1;let s=o.length-n.length;return e===2?t-s:t+s}function gn(r,t){let e=r.commonAncestorContainer.ownerDocument.createNodeIterator(r.commonAncestorContainer,NodeFilter.SHOW_TEXT),i=t===1?r.startContainer:r.endContainer,o=t===1?r.endContainer:r.startContainer,n=e.nextNode();for(;n&&n!==i;)n=e.nextNode();t===2&&(n=e.previousNode());let s=-1,a=()=>{if(n=t===1?e.nextNode():e.previousNode(),n){let h=n.textContent,l=t===1?0:h.length;s=so(h,l,t)}};for(;n&&s===-1&&n!==o;)a();if(n&&s>=0)return{node:n,offset:s};throw new RangeError("No text nodes with non-whitespace text found in range")}function fn(r){if(!r.toString().trim().length)throw new RangeError("Range contains no non-whitespace text");if(r.startContainer.nodeType!==Node.TEXT_NODE)throw new RangeError("Range startContainer is not a text node");if(r.endContainer.nodeType!==Node.TEXT_NODE)throw new RangeError("Range endContainer is not a text node");let t=r.cloneRange(),e=!1,i=!1,o={start:so(r.startContainer.textContent,r.startOffset,1),end:so(r.endContainer.textContent,r.endOffset,2)};if(o.start>=0&&(t.setStart(r.startContainer,o.start),e=!0),o.end>0&&(t.setEnd(r.endContainer,o.end),i=!0),e&&i)return t;if(!e){let{node:n,offset:s}=gn(t,1);n&&s>=0&&t.setStart(n,s)}if(!i){let{node:n,offset:s}=gn(t,2);n&&s>0&&t.setEnd(n,s)}return t}function yn(r){switch(r.nodeType){case Node.ELEMENT_NODE:case Node.TEXT_NODE:return r.textContent?.length??0;default:return 0}}function Sn(r){let t=r.previousSibling,e=0;for(;t;)e+=yn(t),t=t.previousSibling;return e}function _n(r,...t){let e=t.shift(),i=r.ownerDocument.createNodeIterator(r,NodeFilter.SHOW_TEXT),o=[],n=i.nextNode(),s,a=0;for(;e!==void 0&&n;)s=n,a+s.data.length>e?(o.push({node:s,offset:e-a}),e=t.shift()):(n=i.nextNode(),a+=s.data.length);for(;e!==void 0&&s&&a===e;)o.push({node:s,offset:s.data.length}),e=t.shift();if(e!==void 0)throw new RangeError("Offset exceeds text length");return o}var re=class r{constructor(t,e){if(e<0)throw new Error("Offset is invalid");this.element=t,this.offset=e}relativeTo(t){if(!t.contains(this.element))throw new Error("Parent is not an ancestor of current element");let e=this.element,i=this.offset;for(;e!==t;)i+=Sn(e),e=e.parentElement;return new r(e,i)}resolve(t={}){try{return _n(this.element,this.offset)[0]}catch(e){if(this.offset===0&&t.direction!==void 0){let i=document.createTreeWalker(this.element.getRootNode(),NodeFilter.SHOW_TEXT);i.currentNode=this.element;let o=t.direction===1,n=o?i.nextNode():i.previousNode();if(!n)throw e;return{node:n,offset:o?0:n.data.length}}else throw e}}static fromCharOffset(t,e){switch(t.nodeType){case Node.TEXT_NODE:return r.fromPoint(t,e);case Node.ELEMENT_NODE:return new r(t,e);default:throw new Error("Node is not an element or text node")}}static fromPoint(t,e){switch(t.nodeType){case Node.TEXT_NODE:{if(e<0||e>t.data.length)throw new Error("Text node offset is out of range");if(!t.parentElement)throw new Error("Text node has no parent");let i=Sn(t)+e;return new r(t.parentElement,i)}case Node.ELEMENT_NODE:{if(e<0||e>t.childNodes.length)throw new Error("Child node offset is out of range");let i=0;for(let o=0;o<e;o++)i+=yn(t.childNodes[o]);return new r(t,i)}default:throw new Error("Point is not in an element or text node")}}},se=class r{constructor(t,e){this.start=t,this.end=e}relativeTo(t){return new r(this.start.relativeTo(t),this.end.relativeTo(t))}toRange(){let t,e;this.start.element===this.end.element&&this.start.offset<=this.end.offset?[t,e]=_n(this.start.element,this.start.offset,this.end.offset):(t=this.start.resolve({direction:1}),e=this.end.resolve({direction:2}));let i=new Range;return i.setStart(t.node,t.offset),i.setEnd(e.node,e.offset),i}static fromRange(t){let e=re.fromPoint(t.startContainer,t.startOffset),i=re.fromPoint(t.endContainer,t.endOffset);return new r(e,i)}static fromOffsets(t,e,i){return new r(new re(t,e),new re(t,i))}static trimmedRange(t){return fn(r.fromRange(t).toRange())}};var ao=class r{constructor(t,e,i){this.root=t,this.start=e,this.end=i}static fromRange(t,e){let i=se.fromRange(e).relativeTo(t);return new r(t,i.start.offset,i.end.offset)}static fromSelector(t,e){return new r(t,e.start,e.end)}toSelector(){return{type:"TextPositionSelector",start:this.start,end:this.end}}toRange(){return se.fromOffsets(this.root,this.start,this.end).toRange()}},Je=class r{constructor(t,e,i={}){this.root=t,this.exact=e,this.context=i}static fromRange(t,e){let i=t.textContent,o=se.fromRange(e).relativeTo(t),n=o.start.offset,s=o.end.offset,a=32;return new r(t,i.slice(n,s),{prefix:i.slice(Math.max(0,n-a),n),suffix:i.slice(s,Math.min(i.length,s+a))})}static fromSelector(t,e){let{prefix:i,suffix:o}=e;return new r(t,e.exact,{prefix:i,suffix:o})}toSelector(){return{type:"TextQuoteSelector",exact:this.exact,prefix:this.context.prefix,suffix:this.context.suffix}}toRange(t={}){return this.toPositionAnchor(t).toRange()}toPositionAnchor(t={}){let e=this.root.textContent,i=pn(e,this.exact,{...this.context,hint:t.hint});if(!i)throw new Error("Quote not found");return new ao(this.root,i.start,i.end)}};function qs(r){let t=r.tagName.toUpperCase();return t==="IMG"||t==="VIDEO"||t==="AUDIO"||t==="IFRAME"||t==="OBJECT"||t==="EMBED"||t==="CANVAS"}function bn(r,t){let e=t&&_t(t),i=null;if(e)try{i=r.querySelector(e)}catch(o){console.warn(`Invalid cssSelector: ${e}`,o)}return i??r.body??r.documentElement}function vn(r,t){let e;try{e=r.querySelector(t.cssSelector)}catch(o){return console.error(`Invalid domRange cssSelector: ${t.cssSelector}`,o),null}if(!e)return console.error(`Can't resolve domRange cssSelector: ${t.cssSelector}`),null;let i=0;for(let o of e.childNodes)if(o.nodeType===Node.TEXT_NODE){if(i===t.textNodeIndex)return{node:o,offset:t.charOffset};i++}return console.error(`Can't resolve domRange textNodeIndex ${t.textNodeIndex} for selector: ${t.cssSelector}`),null}function ae(r,t){try{let e=t.locations,i=t.text;if(e){let o=Ki(e);if(o){let n=vn(r,o.start),s=o.end?vn(r,o.end):n;if(n&&s)try{let a=r.createRange();return n.offset!==void 0?a.setStart(n.node,n.offset):a.setStartBefore(n.node),s.offset!==void 0?a.setEnd(s.node,s.offset):a.setEndBefore(s.node),a}catch(a){console.warn("Invalid domRange, falling back:",a)}}}if(e){let o=Yi(e);if(o){let n=bn(r,e),s=$i(o,r,n);if(s.length>0)return s[0]}}if(i&&i.highlight){let o=bn(r,e),n=new Je(o,i.highlight,{prefix:i.before,suffix:i.after});try{return n.toRange()}catch{return console.warn("Quote not found:",n),null}}if(e){let o=null;if(!o&&_t(e)&&(o=r.querySelector(_t(e))),!o&&e.fragments){for(let n of e.fragments)if(o=r.getElementById(n),o)break}if(o){let n=r.createRange();return o.childNodes.length===0||qs(o)?(n.selectNode(o),n):(n.setStartBefore(o),n.setEndAfter(o),n)}}}catch(e){console.error(e)}return null}var Q=class{};var Rn="readium-snapper-style",lo="readium-navigation-lock-style",ho=class we extends Q{constructor(){super(...arguments),this.protected=!1,this.locked=!1,this.watchedRanges=[],this.watchedWithoutText=[],this.timelineEntries=new Map,this.cachedFragmentIds=[],this.sortedFragmentIds=[],this.cachedFragmentStarts=new Map}static inDomOrder(t,e,i){let o=t.get(e),n=t.get(i);return!o||!n?0:o.compareDocumentPosition(n)&Node.DOCUMENT_POSITION_FOLLOWING?-1:1}updateTimelineEntries(t,e){this.cachedFragmentIds=t,this.timelineEntries.clear(),this.cachedFragmentStarts.clear();for(let i of t){let o=e.document.getElementById(i);o&&this.timelineEntries.set(i,o)}this.sortedFragmentIds=Array.from(this.timelineEntries.keys()).sort((i,o)=>we.inDomOrder(this.timelineEntries,i,o)),this.refreshFragmentStarts()}refreshFragmentStarts(){for(let[t,e]of this.timelineEntries)this.cachedFragmentStarts.set(t,this.fragmentStart(e))}fragmentStart(t){return this.rectStart(t.getBoundingClientRect())}currentScrollExtent(){return{pos:0,size:0}}rectStart(t){return 0}lineStart(t){return this.rectStart(t)}registerTextLayout(t,e,i){this.textLayoutComms=e,e.register("watch_text_layout",i,(o,n)=>{let s=Array.isArray(o)?o:[];this.watchedRanges=s.map(a=>we.rangeOf(t,a)),this.watchedWithoutText=s.map(a=>!at.deserialize(Array.isArray(a)?a[0]:a)?.highlight),t.requestAnimationFrame(()=>this.sendTextLayout(!0)),n(!0)})}registerNavigationLock(t,e,i){e.register("lock_navigation",i,(o,n)=>{if(this.locked=o===!0,t.document.getElementById(lo)?.remove(),this.locked){let s=t.document.createElement("style");s.dataset.readium="true",s.id=lo,s.textContent=`
                :root {
                    overflow: hidden !important;
                    touch-action: none !important;
                }`,t.document.head.appendChild(s)}n(!0)})}unlockNavigation(t){this.locked=!1,t.document.getElementById(lo)?.remove()}unwatchTextLayout(){this.watchedRanges=[],this.watchedWithoutText=[],this.textLayoutComms=void 0}sendTextLayout(t=!1){if(!this.textLayoutComms?.ready||this.watchedRanges.length===0)return;let e={viewport:this.currentScrollExtent()};t&&(e.pieces=this.watchedRanges.map((i,o)=>{if(!i)return null;let n=this.lineStarts(i);return this.watchedWithoutText[o]?{...n,text:i.toString()}:n})),this.textLayoutComms.send("text_layout",e)}static rangeOf(t,e){let[i,o]=Array.isArray(e)?e:[e];return ae(t.document,new z({href:t.location.href,type:"text/html",text:at.deserialize(i),locations:o?new G({otherLocations:new Map([["cssSelector",o]])}):void 0}))??null}lineStarts(t){let e=t.startContainer.ownerDocument,i={offsets:[],starts:[]},o=e.createTreeWalker(t.commonAncestorContainer,NodeFilter.SHOW_TEXT),n=o.currentNode.nodeType===Node.TEXT_NODE?o.currentNode:o.nextNode(),s=0,a=!1;for(;n;n=o.nextNode()){if(!t.intersectsNode(n)){if(s>0)break;continue}let h=n.textContent??"",l=n===t.startContainer?t.startOffset:0,d=n===t.endContainer?t.endOffset:h.length;if(l>=d)continue;let c=[];for(let C=l;C<d;C++){c.push(s);let v=/\s/.test(h[C]);(!v||!a)&&s++,a=v}let m=C=>{let v=e.createRange();v.setStart(n,C),v.setEnd(n,C+1);let w=v.getClientRects()[0];return w&&(w.width||w.height)?Math.round(this.lineStart(w)):void 0},p=(C,v)=>{i.starts.length>0&&i.starts[i.starts.length-1]===v||(i.offsets.push(c[C-l]),i.starts.push(v))},g=(C,v,w,B)=>{if(v===B)return;if(w-C===1)return p(w,B);let it=C+w>>1,lt=m(it);for(;lt===void 0&&++it<w;)lt=m(it);if(lt===void 0)return p(w,B);g(C,v,it,lt),g(it,lt,w,B)},u=l,f=m(u);for(;f===void 0&&++u<d;)f=m(u);if(f===void 0)continue;let S=d-1,b=m(S);for(;b===void 0&&--S>u;)b=m(S);p(u,f),b!==void 0&&g(u,f,S,b)}return i}nearestPrecedingTimelineEntry(t){let e;for(let i of this.sortedFragmentIds){let o=this.timelineEntries.get(i),n=o.compareDocumentPosition(t);if(o===t||o.contains(t)||(n&Node.DOCUMENT_POSITION_FOLLOWING)!==0)e=i;else break}return e}fragmentFromGeometry(){for(let e of this.sortedFragmentIds){let i=this.timelineEntries.get(e);if(this.inCenterBand(i))return e}let t;for(let e of this.sortedFragmentIds){let i=this.timelineEntries.get(e);if(this.hasScrolledPast(i))t=e;else break}return t}firstFragmentIfReached(){let t=this.sortedFragmentIds[0];if(t===void 0)return;let e=this.timelineEntries.get(t);return this.hasScrolledPast(e)?t:void 0}currentTimelineFragment(){let{pos:t,size:e}=this.currentScrollExtent(),i=t+e/2,o;for(let n of this.sortedFragmentIds){let s=this.cachedFragmentStarts.get(n);if(s!==void 0)if(s<=i)o=n;else break}return o}sortedVisibleFragmentIds(){let{pos:t,size:e}=this.currentScrollExtent(),i=t+e,o=-1,n=-1;for(let s=0;s<this.sortedFragmentIds.length;s++){let a=this.cachedFragmentStarts.get(this.sortedFragmentIds[s]);a!==void 0&&(a<=t&&(o=s),a<i&&(n=s))}return o<0&&(o=0),n<o?[]:this.sortedFragmentIds.slice(o,n+1)}inCenterBand(t){return!1}buildStyles(){return`
        html, body {
            touch-action: manipulation;
            user-select: ${this.protected?"none":"auto"};
        }`}mount(t,e){let i=t.document.createElement("style");return i.dataset.readium="true",i.id=Rn,i.textContent=this.buildStyles(),t.document.head.appendChild(i),e.register("protect",we.moduleName,(o,n)=>{this.protected=!0,i.textContent=this.buildStyles(),n(!0)}),e.register("unprotect",we.moduleName,(o,n)=>{this.protected=!1,i.textContent=this.buildStyles(),n(!0)}),e.log("Snapper Mounted"),!0}unmount(t,e){return t.document.getElementById(Rn)?.remove(),this.timelineEntries.clear(),this.cachedFragmentIds=[],this.sortedFragmentIds=[],this.cachedFragmentStarts.clear(),e.log("Snapper Unmounted"),!0}};ho.moduleName="snapper",ho.CENTER_TOLERANCE=.01;var M=ho;function wn(r){return(r.document.documentElement.dir||r.document.body.dir).toLowerCase()==="rtl"}function co(r){return(r.getComputedStyle(r.document.documentElement).writingMode||r.getComputedStyle(r.document.body).writingMode)==="vertical-lr"}function Js(r){let t=r.getComputedStyle(r.document.documentElement).writingMode||r.getComputedStyle(r.document.body).writingMode;return t==="vertical-rl"||t==="vertical-lr"}var Ze=2147483646;function Ut(r){let t=Js(r),e=t&&co(r),i=r.innerWidth,o=r.innerHeight,n=r.document.scrollingElement,s=n.scrollLeft,a=n.scrollTop,h=parseInt(r.getComputedStyle(r.document.documentElement).getPropertyValue("column-count")),l=t&&!e?n.scrollWidth-i+s:s,d=a;return{isVertical:t,isVertLR:e,viewportInlineSize:t?o:i,viewportBlockSize:t?i:o,pageInlineSize:t?o:i/(h||1),xDocOffset:l,yDocOffset:d,inlineScrollOffset:t?d:l,blockScrollOffset:t?l:d,inlineStart:c=>t?c.top:c.left,blockStart:c=>t?c.left:c.top,inlineSize:c=>t?c.height:c.width,blockSize:c=>t?c.width:c.height,applyPosition(c,m,p,g,u,f){c.style.position="absolute",c.style.zIndex=String(Ze),t?(c.style.top=`${m*f}px`,c.style.left=`${p*f}px`,c.style.height=`${g*f}px`,c.style.width=`${u*f}px`):(c.style.left=`${m*f}px`,c.style.top=`${p*f}px`,c.style.width=`${g*f}px`,c.style.height=`${u*f}px`)},toRect(c,m,p,g){return t?new DOMRect(m,c,g,p):new DOMRect(c,m,p,g)}}}function mo(r){return parseInt(r.getComputedStyle(r.document.documentElement).getPropertyValue("column-count"))}function Zs(r){let t=getComputedStyle(r),e=parseFloat(t.paddingTop||"0"),i=parseFloat(t.paddingBottom||"0");return r.clientHeight-e-i}function uo(r){let t=mo(r);if(!t)return!1;let e=r.document.querySelectorAll("div[id^='readium-virtual-page']");for(let h of e)h.remove();let i=e.length,o=r.document.scrollingElement.scrollWidth,n=r.visualViewport.width,s=Math.round(o/n*t)%t,a=t===1||s===0?0:t-s;if(a>0)for(let h=0;h<a;h++){let l=r.document.createElement("div");l.setAttribute("id",`readium-virtual-page-${h}`),l.dataset.readium="true",CSS.supports("break-before","column")?l.style.breakBefore="column":(CSS.supports("break-inside","avoid-column")&&(l.style.breakInside="avoid-column"),l.style.height=Zs(r.document.documentElement)+"px"),l.innerHTML="&#8203;",r.document.body.appendChild(l)}return i!==a}function le(r){let t=r.document.createElement("style");t.appendChild(r.document.createTextNode("*{}")),r.document.body.appendChild(t),r.document.body.removeChild(t)}var Qe=()=>typeof window.CSSTransformValue<"u",Qs=new Set(["backgroundColor","textColor","linkColor","visitedColor","primaryColor","secondaryColor","selectionBackgroundColor","selectionTextColor","blendFilter","darkenFilter","invertFilter","invertGaiji"]),Cn=/--(?:USER|RS)__([\w-]+)/g;function En(r,t){let e=r??"",i=t??"",o=new Set;for(let n of e.matchAll(Cn))o.add(n[1]);for(let n of i.matchAll(Cn))o.add(n[1]);for(let n of o)if(!Qs.has(n))return!0;return!1}function Ce(r){let t={},e=r.document.documentElement.style;for(let i in r.document.documentElement.style)Object.hasOwn(e,i)&&!Number.isNaN(Number.parseInt(i))&&(t[e[i]]=e.getPropertyValue(e[i]));return t}function ti(r,t){let e=Ce(r);Object.keys(e).forEach(i=>{t.hasOwnProperty(i)||It(r,i)}),Object.entries(t).forEach(([i,o])=>{e[i]!==o&&Rt(r,i,o)})}function po(r,t){return r.document.documentElement.style.getPropertyValue(t)}function Rt(r,t,e){r.document.documentElement.style.setProperty(t,e)}function It(r,t){r.document.documentElement.style.removeProperty(t)}function Pn(r){return r<.5?2*r*r:-1+(4-2*r)*r}function F(r){let t=r.getSelection();t&&t.removeAllRanges()}var ta=["a","area","audio","button","canvas","details","input","label","option","select","submit","textarea","video"],ea=["dialog","radiogroup","radio","menu","menuitem"];function Ee(r){return ia(r)?null:go(r)?r:r.parentElement?Ee(r.parentElement):null}function ia(r){return r?r.closest("[inert]")!==null||r.hasAttribute("disabled"):!0}function go(r){return r?r.role&&ea.includes(r.role)?!0:r.tagName.toLowerCase()==="iframe"?!1:r.tabIndex>=0?!0:ta.includes(r.nodeName.toLowerCase())||r.hasAttribute("contenteditable")&&r.getAttribute("contenteditable")?.toLowerCase()!=="false":!1}function wt(r,t){let e=Fn(r,r.document.body,t),i=r._readium_cssSelectorGenerator.getCssSelector(e,{selectors:["tag","id","class","nthchild","nthoftype","attribute"]});return new z({href:"#",type:"application/xhtml+xml",locations:new G({otherLocations:new Map([["cssSelector",i]])}),text:new at({highlight:e.textContent||void 0})})}function Fn(r,t,e){for(var i=0;i<t.children.length;i++){let o=t.children[i];if(!ra(o)&&oa(r,o,e))return na(r,o)?o:Fn(r,o,e)}return t}function oa(r,t,e){if(t===document.body||t===document.documentElement)return!0;if(!document||!document.documentElement||!document.body)return!1;let i=t.getBoundingClientRect();return e?i.bottom>0&&i.top<r.innerHeight:i.right>0&&i.left<r.innerWidth}function na(r,t){let e=t.getBoundingClientRect();return e.top>=0&&e.left>=0&&e.bottom<=r.innerHeight&&e.right<=r.innerWidth}function ra(r){let t=getComputedStyle(r);if(t){let e=t.getPropertyValue("display");if(e!="block"&&e!="list-item"||t.getPropertyValue("opacity")==="0")return!0}return!1}var xn={maxVelocity:200,minVariance:.01,historySize:20,minDirectionChanges:.2,maxConsistentScrolls:15},he={maxVelocity:200,minVariance:1e-5,historySize:100,minDirectionChanges:.1,maxConsistentScrolls:20},ei={maxSelectionsPerSecond:500,minVariance:50,historySize:20},kn={enabled:!0,maxSelectionPercent:.1,minThreshold:100,absoluteMaxChars:5e3,historySize:20};var pt=class{constructor(t={}){this.history=[],this.consistentScrollCount=0,this.options={...xn,...t}}analyze(t,e,i){if(i<=0)return!1;let o=Math.abs(e)/i,n=Date.now();if(this.history.push({timestamp:n,direction:t,velocity:o,distance:Math.abs(e)}),this.history=this.history.filter(g=>n-g.timestamp<2e3).slice(-(this.options.historySize||20)),this.history.length<3)return!1;if(o>this.options.maxVelocity)return this.resetAfterDetection(),!0;let s=this.history.map(g=>g.velocity),a=this.history.map(g=>g.distance),h=s.reduce((g,u)=>g+u,0)/s.length,l=a.reduce((g,u)=>g+u,0)/a.length,d=s.reduce((g,u)=>g+Math.pow(u-h,2),0)/s.length,c=a.reduce((g,u)=>g+Math.pow(u-l,2),0)/a.length;if(d<this.options.minVariance&&c<l*.1){if(this.consistentScrollCount++,this.consistentScrollCount>=(this.options.maxConsistentScrolls||10))return this.resetAfterDetection(),!0}else this.consistentScrollCount=Math.max(0,this.consistentScrollCount-1);let m=0,p=this.history[0].direction;for(let g=1;g<this.history.length;g++)this.history[g].direction!==p&&(m++,p=this.history[g].direction);return m/this.history.length>(this.options.minDirectionChanges||.3)?(this.resetAfterDetection(),!0):!1}resetAfterDetection(){this.history=this.history.slice(-3),this.consistentScrollCount=0}clear(){this.history=[],this.consistentScrollCount=0}};var Tn="readium-column-snapper-style",sa=200,zn=class H extends M{constructor(){super(...arguments),this.isSnapProtectionEnabled=!1,this.patternAnalyzer=null,this.lastTurnTime=0,this.rtl=!1,this.shakeTimeout=0,this.snappingCancelled=!1,this.alreadyScrollLeft=0,this.overscroll=0,this.cachedScrollWidth=0,this.touchState=0,this.startingX=void 0,this.endingX=void 0,this.onTouchStarter=this.onTouchStart.bind(this),this.onTouchEnder=this.onTouchEnd.bind(this),this.onWidthChanger=this.onWidthChange.bind(this),this.onTouchMover=this.onTouchMove.bind(this)}doc(){return this.wnd.document.scrollingElement}scrollOffset(){let t=this.doc().scrollLeft;return t!==0?t:this.alreadyScrollLeft}snapOffset(t){let e=t+(this.rtl?-1:1);return e-e%this.wnd.innerWidth}snapNormOffset(t){let e=t+1;return e-e%this.wnd.innerWidth}normScroll(){let t=this.doc().scrollLeft||(this.doc().style.transform?this.alreadyScrollLeft:0);return this.rtl?Math.abs(t):Math.max(0,this.wnd.scrollX>0?this.wnd.scrollX:t)}currentScrollExtent(){return{pos:this.normScroll(),size:this.wnd.innerWidth}}rectStart(t){return this.normScroll()+(this.rtl?this.wnd.innerWidth-t.right:t.left)}lineStart(t){let e=this.rectStart(t);return e-e%this.wnd.innerWidth}hasScrolledPast(t){let e=t.getBoundingClientRect();return this.rtl?e.left>=this.wnd.innerWidth:e.right<=0}currentTimelineFragment(){if(this.timelineEntries.size===0)return;let t=this.wnd.innerWidth,e;for(let i of this.sortedFragmentIds){let o=this.timelineEntries.get(i).getBoundingClientRect();if(this.rtl?o.right>0:o.left<t)e=i;else break}return e}sortedVisibleFragmentIds(){let t=this.wnd.innerWidth;return this.sortedFragmentIds.filter(e=>{let i=this.timelineEntries.get(e).getBoundingClientRect();return this.rtl?i.left<t&&i.right>0:i.right>0&&i.left<t})}reportProgress(t){let e=this.cachedScrollWidth,i=this.wnd.innerWidth,o=this.normScroll(),n=Math.max(1,e-i),s=Math.max(0,Math.min(1,o/n)),a=Math.max(0,Math.min(1,(o+i)/e));this.comms.send("progress",{start:s,end:a,fragmentId:t!==void 0?t:this.currentTimelineFragment(),visibleFragmentIds:this.sortedVisibleFragmentIds()}),this.sendTextLayout()}shake(){if(this.overscroll!==0||this.shakeTimeout!==0)return;let t=this.doc(),e=this.normScroll()<5,i=this.rtl?e?"readium-bounce-r":"readium-bounce-l":"readium-bounce-r";t.classList.add(i);let o=this.scrollOffset();this.shakeTimeout=this.wnd.setTimeout(()=>{t.classList.remove(i),this.shakeTimeout=0,this.doc().scrollLeft=o},150)}takeOverSnap(){this.snappingCancelled=!0,this.clearTouches();let t=this.doc();this.overscroll=t.style.transform?.length>12?parseFloat(t.style.transform.slice(12).split("px")[0]):0}snapCurrentOffset(t=!1,e=!1){let i=this.doc(),o=mo(this.wnd),n=this.cachedScrollWidth-this.wnd.innerWidth,s=Math.min(Math.max(0,this.normScroll()),n),a=this.dragOffset(),h=this.rtl?-a:a,l=this.wnd.innerWidth/3*(h>0?2:1),d=Math.min(n,Math.max(0,this.snapNormOffset(s+l))),c=this.rtl?-d:d,m=this.rtl?-s:s,p=c>m?"right":"left";if(this.checkSuspiciousSnap(p,Math.abs(c-m)),t&&c!==m&&this.wnd.parent?.document?.documentElement?.dataset?.motion!=="reduced"){this.snappingCancelled=!1;let g=(b,C,v,w)=>v>w?C:b+(C-b)*Pn(v/w),u=sa*o,f,S=b=>{if(this.snappingCancelled)return;f||(f=b);let C=b-f,v=g(this.overscroll,0,C,u),w=g(m,c,C,u);i.scrollLeft=w,this.overscroll!==0&&(Qe()?i.attributeStyleMap.set("transform",new CSSTransformValue([new CSSTranslate(CSS.px(-v),CSS.px(0),CSS.px(0))])):i.style.transform=`translate3d(${-v}px, 0px, 0px)`),C<u?this.wnd.requestAnimationFrame(S):(this.clearTouches(),i.style.removeProperty("transform"),i.scrollLeft=c,e||this.reportProgress())};this.wnd.requestAnimationFrame(S)}else i.style.removeProperty("transform"),this.wnd.requestAnimationFrame(()=>{i.scrollLeft=c,this.clearTouches(),e||this.reportProgress()})}dragOffset(){return(this.startingX??0)-(this.endingX??0)}clearTouches(){this.startingX=void 0,this.endingX=void 0,this.overscroll=0}onTouchStart(t){if(t.stopPropagation(),!this.locked){switch(this.takeOverSnap(),t.touches.length){case 1:break;case 2:this.onTouchEnd(t);return;default:{this.onTouchEnd(t),this.comms.send("tap_more",t.touches.length);return}}this.startingX=t.touches[0].clientX,this.alreadyScrollLeft=this.doc().scrollLeft,this.touchState=1}}onTouchEnd(t){if(this.touchState===2){let e=this.dragOffset(),i=this.normScroll(),o=this.cachedScrollWidth-this.wnd.innerWidth,n=this.rtl?-e:e;this.cachedScrollWidth<=this.wnd.innerWidth?(this.reportProgress(),n>5&&this.comms.send("no_more",void 0),n<-5&&this.comms.send("no_less",void 0)):i<5&&n<5?(this.alreadyScrollLeft=0,this.comms.send("no_less",void 0)):o-i<5&&n>5&&(this.alreadyScrollLeft=this.rtl?-o:o,this.comms.send("no_more",void 0)),this.snapCurrentOffset(!0),this.comms.send("swipe",e)}this.touchState=0}onWidthChange(){this.cachedScrollWidth=this.doc().scrollWidth,this.comms.ready&&(this.snapCurrentOffset(),this.sendTextLayout(!0))}onTouchMove(t){if(this.touchState===0)return;this.touchState===1&&(this.touchState=2,F(this.wnd)),this.endingX=t.touches[0].clientX;let e=this.dragOffset(),i=this.alreadyScrollLeft+e,o=this.rtl?-(this.cachedScrollWidth-this.wnd.innerWidth):0,n=this.rtl?0:this.cachedScrollWidth-this.wnd.innerWidth;i<o?(this.overscroll=i,Qe()?this.doc().attributeStyleMap.set("transform",new CSSTransformValue([new CSSTranslate(CSS.px(-i),CSS.px(0),CSS.px(0))])):this.doc().style.transform=`translate3d(${-i}px, 0px, 0px)`):i>n?(this.overscroll=i,Qe()?this.doc().attributeStyleMap.set("transform",new CSSTransformValue([new CSSTranslate(CSS.px(-i),CSS.px(0),CSS.px(0))])):this.doc().style.transform=`translate3d(${-i}px, 0px, 0px)`):(this.overscroll=0,this.doc().style.removeProperty("transform"),this.doc().scrollLeft=i)}enableSnapProtection(){this.patternAnalyzer||(this.patternAnalyzer=new pt({maxVelocity:this.wnd.innerWidth,minVariance:.1,historySize:5,maxConsistentScrolls:3,minDirectionChanges:.3}),this.isSnapProtectionEnabled=!0,this.comms?.log("Snap protection enabled"))}checkSuspiciousSnap(t,e){if(!this.isSnapProtectionEnabled||!this.patternAnalyzer)return;let i=Date.now(),o=i-(this.lastTurnTime||i);this.lastTurnTime=i,this.patternAnalyzer.analyze(t,e,o)&&this.comms?.send("content_protection",{type:"suspicious_snapping",timestamp:Date.now(),event:null})}mount(t,e){if(this.wnd=t,this.comms=e,this.rtl=wn(t),!super.mount(t,e))return!1;t.navigator.epubReadingSystem&&(t.navigator.epubReadingSystem.layoutStyle="paginated");let i=t.document.createElement("style");i.dataset.readium="true",i.id=Tn,i.textContent=`
        @keyframes readium-bounce-l-animation {
            0%, 100% {transform: translate3d(0, 0, 0);}
            50% {transform: translate3d(-50px, 0, 0);}
        }

        @keyframes readium-bounce-r-animation {
            0%, 100% {transform: translate3d(0, 0, 0);}
            50% {transform: translate3d(50px, 0, 0);}
        }

        .readium-bounce-l {
            animation: readium-bounce-l-animation 150ms ease-out 1;
        }

        .readium-bounce-r {
            animation: readium-bounce-r-animation 150ms ease-out 1;
        }

        html {
            overflow: hidden;
        }

        body {
            -ms-overflow-style: none; /* for Internet Explorer, Edge */
        }

        * {
            scrollbar-width: none; /* for Firefox */
        }

        body::-webkit-scrollbar {
            display: none; /* for Chrome, Safari, and Opera */
        }
        `,t.document.head.appendChild(i),this.resizeObserver=new ResizeObserver(()=>{t.requestAnimationFrame(()=>{t&&uo(t)}),this.onWidthChange()}),this.resizeObserver.observe(t.document.body),this.mutationObserver=new MutationObserver(s=>{for(let a of s)if(a.target===this.wnd.document.documentElement){let h=a.oldValue,l=a.target.getAttribute("style"),d=/transform\s*:\s*([^;]+)/,c=h?.match(d),m=l?.match(d);(!c&&!m&&En(h,l)||c&&!m||c&&m&&c[1]!==m[1])&&(t.requestAnimationFrame(()=>{t&&uo(t)}),this.onWidthChange())}else t.requestAnimationFrame(()=>this.cachedScrollWidth=this.doc().scrollWidth)}),t.frameElement&&this.mutationObserver.observe(t.frameElement,{attributes:!0,attributeFilter:["style"]}),this.mutationObserver.observe(t.document,{attributes:!0,attributeFilter:["style"]}),this.mutationObserver.observe(t.document.documentElement,{attributes:!0,attributeFilter:["style"]}),e.register("scroll_protection",H.moduleName,(s,a)=>{this.enableSnapProtection(),a(!0)});let o=s=>{let a=this.doc().scrollLeft;return this.doc().scrollLeft=this.snapOffset(s),a!==this.doc().scrollLeft},n=s=>{let a=this.doc().scrollLeft,h=this.snapNormOffset(Math.max(0,Math.min(this.cachedScrollWidth-t.innerWidth,s)));return this.doc().scrollLeft=-h,a!==this.doc().scrollLeft};return t.addEventListener("orientationchange",this.onWidthChanger),t.addEventListener("resize",this.onWidthChanger),t.requestAnimationFrame(()=>this.cachedScrollWidth=this.doc().scrollWidth),this.registerTextLayout(t,e,H.moduleName),this.registerNavigationLock(t,e,H.moduleName),e.register("go_progression",H.moduleName,(s,a)=>{let h=s;if(h<0||h>1){e.send("error",{message:"go_progression must be given a position from 0.0 to 1.0"}),a(!1);return}this.wnd.requestAnimationFrame(()=>{this.cachedScrollWidth=this.doc().scrollWidth;let l=(this.cachedScrollWidth-t.innerWidth)*h;this.rtl?this.doc().scrollLeft=-this.snapNormOffset(l):this.doc().scrollLeft=this.snapOffset(l),this.reportProgress(),F(this.wnd),a(!0)})}),e.register("go_id",H.moduleName,(s,a)=>{let h=t.document.getElementById(s);if(!h){a(!1);return}this.wnd.requestAnimationFrame(()=>{this.rtl?this.doc().scrollLeft=-this.snapNormOffset(this.rectStart(h.getBoundingClientRect())):this.doc().scrollLeft=this.snapOffset(h.getBoundingClientRect().left+t.scrollX);let l=s;this.reportProgress(this.timelineEntries.has(l)?l:this.nearestPrecedingTimelineEntry(h)),F(this.wnd),a(!0)})}),e.register("go_text",H.moduleName,(s,a)=>{let h=M.rangeOf(t,s);if(!h){a(!1);return}this.wnd.requestAnimationFrame(()=>{this.rtl?this.doc().scrollLeft=-this.snapNormOffset(this.rectStart(h.getBoundingClientRect())):this.doc().scrollLeft=this.snapOffset(h.getBoundingClientRect().left+t.scrollX),this.reportProgress(this.nearestPrecedingTimelineEntry(h.startContainer)),F(this.wnd),a(!0)})}),e.register("go_end",H.moduleName,(s,a)=>{this.wnd.requestAnimationFrame(()=>{this.cachedScrollWidth=this.doc().scrollWidth;let h;if(this.rtl?h=-this.snapNormOffset(this.cachedScrollWidth-t.innerWidth):h=this.snapOffset(this.cachedScrollWidth),this.doc().scrollLeft===h)return a(!1);this.doc().scrollLeft=h,this.reportProgress(this.sortedFragmentIds[this.sortedFragmentIds.length-1]),F(this.wnd),a(!0)})}),e.register("go_start",H.moduleName,(s,a)=>{this.wnd.requestAnimationFrame(()=>{if(this.doc().scrollLeft===0)return a(!1);this.doc().scrollLeft=0,this.reportProgress(this.firstFragmentIfReached()),F(this.wnd),a(!0)})}),e.register("go_prev",H.moduleName,(s,a)=>{this.wnd.requestAnimationFrame(()=>{this.cachedScrollWidth=this.doc().scrollWidth;let h;this.rtl?h=n(this.normScroll()-t.innerWidth):h=o(t.scrollX-t.innerWidth),this.reportProgress(),h&&(F(this.wnd),this.checkSuspiciousSnap("left",this.wnd.innerWidth)),a(h)})}),e.register("go_next",H.moduleName,(s,a)=>{this.wnd.requestAnimationFrame(()=>{this.cachedScrollWidth=this.doc().scrollWidth;let h;this.rtl?h=n(this.normScroll()+t.innerWidth):h=o(t.scrollX+t.innerWidth),this.reportProgress(),h&&(F(this.wnd),this.checkSuspiciousSnap("right",this.wnd.innerWidth)),a(h)})}),e.register("unfocus",H.moduleName,(s,a)=>{this.snappingCancelled=!0,F(this.wnd),a(!0)}),e.register("shake",H.moduleName,(s,a)=>{this.shake(),a(!0)}),e.register("focus",H.moduleName,(s,a)=>{this.wnd.requestAnimationFrame(()=>{this.cachedScrollWidth=this.doc().scrollWidth,this.snapCurrentOffset(!1,!0),this.reportProgress(),a(!0)})}),e.register("first_visible_locator",H.moduleName,(s,a)=>{let h=wt(t,!1);this.comms.send("first_visible_locator",h.serialize()),a(!0)}),e.register("timeline_entries",H.moduleName,(s,a)=>{this.updateTimelineEntries(Array.isArray(s)?s:[],t),t.requestAnimationFrame(()=>this.reportProgress()),a(!0)}),t.addEventListener("touchstart",this.onTouchStarter,{passive:!0}),t.addEventListener("touchend",this.onTouchEnder,{passive:!0}),t.addEventListener("touchmove",this.onTouchMover,{passive:!0}),t.document.addEventListener("touchstart",()=>{}),e.log("ColumnSnapper Mounted"),!0}unmount(t,e){return this.snappingCancelled=!0,e.unregisterAll(H.moduleName),this.unwatchTextLayout(),this.unlockNavigation(t),this.resizeObserver.disconnect(),this.mutationObserver.disconnect(),this.patternAnalyzer&&(this.patternAnalyzer.clear(),this.patternAnalyzer=null,this.isSnapProtectionEnabled=!1),t.removeEventListener("touchstart",this.onTouchStarter),t.removeEventListener("touchend",this.onTouchEnder),t.removeEventListener("touchmove",this.onTouchMover),t.removeEventListener("orientationchange",this.onWidthChanger),t.removeEventListener("resize",this.onWidthChanger),t.document.getElementById(Tn)?.remove(),this.timelineEntries.clear(),e.log("ColumnSnapper Unmounted"),super.unmount(t,e)}};zn.moduleName="column_snapper";var Ln=zn;var On="readium-scroll-snapper-style",Mn=class V extends M{constructor(){super(...arguments),this.patternAnalyzer=null,this.lastScrollTime=0,this.isScrollProtectionEnabled=!1,this.initialScrollHandled=!1,this.isScrolling=!1,this.lastScrollTop=0,this.isResizing=!1,this.resizeDebounce=null,this.handleScroll=t=>{if(this.comms.ready&&!this.isResizing){if(!this.initialScrollHandled){this.lastScrollTop=this.doc().scrollTop,this.initialScrollHandled=!0,this.reportProgress();return}this.isScrolling||(this.isScrolling=!0,this.wnd.requestAnimationFrame(()=>{this.reportProgress();let e=this.doc().scrollTop,i=e-this.lastScrollTop;if(this.lastScrollTop=e,this.isScrollProtectionEnabled&&Math.abs(i)>5){let o=Date.now(),n=o-(this.lastScrollTime||o);if(this.patternAnalyzer&&this.patternAnalyzer.analyze(i>0?"down":"up",Math.abs(i),n)){let s=t.target&&"tagName"in t.target?{tagName:t.target.tagName}:null;this.comms?.send("content_protection",{type:"suspicious_scrolling",timestamp:Date.now(),scrollDelta:i,scrollDirection:i>0?"down":"up",targetElement:s})}this.lastScrollTime=o}this.comms.send("scroll",i),this.isScrolling=!1}))}}}doc(){return this.wnd.document.scrollingElement}hasScrolledPast(t){let e=this.wnd.innerHeight/2;return t.getBoundingClientRect().top<=e+this.wnd.innerHeight*M.CENTER_TOLERANCE}inCenterBand(t){let e=t.getBoundingClientRect(),i=this.wnd.innerHeight/2,o=this.wnd.innerHeight*M.CENTER_TOLERANCE,n=i-o,s=i+o;return e.height===0?e.top<=s&&e.top>=n:Math.max(0,Math.min(e.bottom,s)-Math.max(e.top,n))/e.height>=M.CENTER_TOLERANCE}currentScrollExtent(){return{pos:this.doc().scrollTop,size:this.wnd.innerHeight}}rectStart(t){return t.top+this.doc().scrollTop}reportProgress(t){if(!this.comms.ready)return;let e=Math.ceil(this.doc().scrollTop),i=this.doc().scrollHeight,o=this.wnd.innerHeight,n=Math.max(0,Math.min(1,e/i)),s=Math.max(0,Math.min(1,(e+o)/i));this.comms.send("progress",{start:n,end:s,fragmentId:t!==void 0?t:this.currentTimelineFragment(),visibleFragmentIds:this.sortedVisibleFragmentIds()}),this.sendTextLayout()}enableScrollProtection(){this.patternAnalyzer||(this.patternAnalyzer=new pt(he),this.isScrollProtectionEnabled=!0,this.comms?.log("Scroll protection enabled"))}mount(t,e){this.wnd=t,this.comms=e,this.initialScrollHandled=!1,this.lastScrollTop=0,this.isResizing=!1,this.resizeDebounce&&(this.wnd.clearTimeout(this.resizeDebounce),this.resizeDebounce=null),t.navigator.epubReadingSystem&&(t.navigator.epubReadingSystem.layoutStyle="scrolling");let i=t.document.createElement("style");return i.dataset.readium="true",i.id=On,i.textContent=`
        * {
            scrollbar-width: none; /* for Firefox */
        }

        body::-webkit-scrollbar {
            display: none; /* for Chrome, Safari, and Opera */
        }
        `,t.document.head.appendChild(i),this.resizeObserver=new ResizeObserver(()=>{this.resizeDebounce&&this.wnd.clearTimeout(this.resizeDebounce),this.isResizing=!0,this.resizeDebounce=this.wnd.setTimeout(()=>{this.isResizing=!1,this.resizeDebounce=null,this.refreshFragmentStarts(),this.reportProgress(),this.sendTextLayout(!0)},50)}),this.resizeObserver.observe(t.document.body),t.addEventListener("scroll",this.handleScroll,{passive:!0}),e.register("force_webkit_recalc",V.moduleName,()=>{le(this.wnd);let o=this.doc().scrollTop;o>1?this.doc().scrollTop=o-1:this.doc().scrollTop=o+1,this.doc().scrollTop=o}),this.registerTextLayout(t,e,V.moduleName),this.registerNavigationLock(t,e,V.moduleName),e.register("go_progression",V.moduleName,(o,n)=>{let s=o;if(s<0||s>1){e.send("error",{message:"go_progression must be given a position from 0.0 to 1.0"}),n(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=this.doc().offsetHeight*s,this.reportProgress(this.fragmentFromGeometry()),F(this.wnd),n(!0)})}),e.register("go_id",V.moduleName,(o,n)=>{let s=t.document.getElementById(o);if(!s){n(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=s.getBoundingClientRect().top+t.scrollY-t.innerHeight/2;let a=o;this.reportProgress(this.timelineEntries.has(a)?a:this.nearestPrecedingTimelineEntry(s)),F(this.wnd),n(!0)})}),e.register("go_text",V.moduleName,(o,n)=>{let s=M.rangeOf(t,o);if(!s){n(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=s.getBoundingClientRect().top+t.scrollY-t.innerHeight/2,this.reportProgress(this.nearestPrecedingTimelineEntry(s.startContainer)),F(this.wnd),n(!0)})}),e.register("go_start",V.moduleName,(o,n)=>{if(this.doc().scrollTop===0)return n(!1);this.doc().scrollTop=0,this.reportProgress(this.firstFragmentIfReached()),n(!0)}),e.register("go_end",V.moduleName,(o,n)=>{if(this.doc().scrollTop===this.doc().scrollHeight-this.doc().offsetHeight)return n(!1);this.doc().scrollTop=this.doc().scrollHeight-this.doc().offsetHeight,this.reportProgress(this.sortedFragmentIds[this.sortedFragmentIds.length-1]),n(!0)}),e.register("unfocus",V.moduleName,(o,n)=>{F(this.wnd),n(!0)}),e.register("scroll_protection",V.moduleName,(o,n)=>{this.enableScrollProtection(),n(!0)}),e.register(["go_next","go_prev"],V.moduleName,(o,n)=>n(!1)),e.register("focus",V.moduleName,(o,n)=>{this.reportProgress(),n(!0)}),e.register("first_visible_locator",V.moduleName,(o,n)=>{let s=wt(t,!0);this.comms.send("first_visible_locator",s.serialize()),n(!0)}),e.register("timeline_entries",V.moduleName,(o,n)=>{this.updateTimelineEntries(Array.isArray(o)?o:[],t),n(!0)}),e.log("ScrollSnapper Mounted"),!0}unmount(t,e){return e.unregisterAll(V.moduleName),this.unwatchTextLayout(),this.unlockNavigation(t),this.resizeObserver.disconnect(),this.handleScroll&&t.removeEventListener("scroll",this.handleScroll),t.document.getElementById(On)?.remove(),this.timelineEntries.clear(),this.cachedFragmentIds=[],this.sortedFragmentIds=[],this.cachedFragmentStarts.clear(),this.patternAnalyzer&&(this.patternAnalyzer.clear(),this.patternAnalyzer=null,this.isScrollProtectionEnabled=!1),e.log("ScrollSnapper Unmounted"),!0}};Mn.moduleName="scroll_snapper";var Nn=Mn;var An=class X extends M{constructor(){super(...arguments),this.patternAnalyzer=null,this.lastScrollTime=0,this.isScrollProtectionEnabled=!1,this.initialScrollHandled=!1,this.isScrolling=!1,this.lastScrollTop=0,this.isResizing=!1,this.resizeDebounce=null,this.handleScroll=t=>{if(this.comms.ready&&!this.isResizing){if(!this.initialScrollHandled){this.lastScrollTop=this.doc().scrollTop,this.initialScrollHandled=!0,this.reportProgress();return}this.isScrolling||(this.isScrolling=!0,this.wnd.requestAnimationFrame(()=>{this.reportProgress();let e=this.doc().scrollTop,i=e-this.lastScrollTop;if(this.lastScrollTop=e,this.isScrollProtectionEnabled&&Math.abs(i)>5){let o=Date.now(),n=o-(this.lastScrollTime||o);if(this.patternAnalyzer&&this.patternAnalyzer.analyze(i>0?"down":"up",Math.abs(i),n)){let s=t.target&&"tagName"in t.target?{tagName:t.target.tagName}:null;this.comms?.send("content_protection",{type:"suspicious_scrolling",timestamp:Date.now(),scrollDelta:i,scrollDirection:i>0?"down":"up",targetElement:s})}this.lastScrollTime=o}this.comms.send("scroll",i),this.isScrolling=!1}))}}}doc(){return this.wnd.document.scrollingElement}hasScrolledPast(t){let e=this.wnd.innerHeight/2;return t.getBoundingClientRect().top<=e+this.wnd.innerHeight*M.CENTER_TOLERANCE}inCenterBand(t){let e=t.getBoundingClientRect(),i=this.wnd.innerHeight/2,o=this.wnd.innerHeight*M.CENTER_TOLERANCE,n=i-o,s=i+o;return e.height===0?e.top<=s&&e.top>=n:Math.max(0,Math.min(e.bottom,s)-Math.max(e.top,n))/e.height>=M.CENTER_TOLERANCE}currentScrollExtent(){return{pos:this.doc().scrollTop,size:this.wnd.innerHeight}}rectStart(t){return t.top+this.doc().scrollTop}reportProgress(t){if(!this.comms.ready)return;let e=Math.ceil(this.doc().scrollTop),i=this.doc().scrollHeight,o=this.wnd.innerHeight,n=Math.max(0,Math.min(1,e/i)),s=Math.max(0,Math.min(1,(e+o)/i));this.comms.send("progress",{start:n,end:s,fragmentId:t!==void 0?t:this.currentTimelineFragment(),visibleFragmentIds:this.sortedVisibleFragmentIds()}),this.sendTextLayout()}enableScrollProtection(){this.patternAnalyzer||(this.patternAnalyzer=new pt(he),this.isScrollProtectionEnabled=!0,this.comms?.log("Scroll protection enabled"))}mount(t,e){return this.wnd=t,this.comms=e,this.initialScrollHandled=!1,this.lastScrollTop=0,this.isResizing=!1,this.resizeDebounce&&(this.wnd.clearTimeout(this.resizeDebounce),this.resizeDebounce=null),this.resizeObserver=new ResizeObserver(()=>{this.resizeDebounce&&this.wnd.clearTimeout(this.resizeDebounce),this.isResizing=!0,this.resizeDebounce=this.wnd.setTimeout(()=>{this.isResizing=!1,this.resizeDebounce=null,this.refreshFragmentStarts(),this.reportProgress(),this.sendTextLayout(!0)},50)}),this.resizeObserver.observe(t.document.body),t.addEventListener("scroll",this.handleScroll,{passive:!0}),e.register("force_webkit_recalc",X.moduleName,()=>{le(this.wnd);let i=this.doc().scrollTop;i>1?this.doc().scrollTop=i-1:this.doc().scrollTop=i+1,this.doc().scrollTop=i}),this.registerTextLayout(t,e,X.moduleName),this.registerNavigationLock(t,e,X.moduleName),e.register("go_progression",X.moduleName,(i,o)=>{let n=i;if(n<0||n>1){e.send("error",{message:"go_progression must be given a position from 0.0 to 1.0"}),o(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=this.doc().offsetHeight*n,this.reportProgress(this.fragmentFromGeometry()),F(this.wnd),o(!0)})}),e.register("go_id",X.moduleName,(i,o)=>{let n=t.document.getElementById(i);if(!n){o(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=n.getBoundingClientRect().top+t.scrollY-t.innerHeight/2;let s=i;this.reportProgress(this.timelineEntries.has(s)?s:this.nearestPrecedingTimelineEntry(n)),F(this.wnd),o(!0)})}),e.register("go_text",X.moduleName,(i,o)=>{let n=M.rangeOf(t,i);if(!n){o(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollTop=n.getBoundingClientRect().top+t.scrollY-t.innerHeight/2,this.reportProgress(this.nearestPrecedingTimelineEntry(n.startContainer)),F(this.wnd),o(!0)})}),e.register("go_start",X.moduleName,(i,o)=>{if(this.doc().scrollTop===0)return o(!1);this.doc().scrollTop=0,this.reportProgress(this.firstFragmentIfReached()),o(!0)}),e.register("go_end",X.moduleName,(i,o)=>{if(this.doc().scrollTop===this.doc().scrollHeight-this.doc().offsetHeight)return o(!1);this.doc().scrollTop=this.doc().scrollHeight-this.doc().offsetHeight,this.reportProgress(this.sortedFragmentIds[this.sortedFragmentIds.length-1]),o(!0)}),e.register("unfocus",X.moduleName,(i,o)=>{F(this.wnd),o(!0)}),e.register("scroll_protection",X.moduleName,(i,o)=>{this.enableScrollProtection(),o(!0)}),e.register(["go_next","go_prev"],X.moduleName,(i,o)=>o(!1)),e.register("focus",X.moduleName,(i,o)=>{this.reportProgress(),o(!0)}),e.register("first_visible_locator",X.moduleName,(i,o)=>{let n=wt(t,!0);e.send("first_visible_locator",n.serialize()),o(!0)}),e.register("timeline_entries",X.moduleName,(i,o)=>{this.updateTimelineEntries(Array.isArray(i)?i:[],t),o(!0)}),e.log("WebPubSnapper Mounted"),!0}unmount(t,e){return e.unregisterAll(X.moduleName),this.unwatchTextLayout(),this.unlockNavigation(t),this.resizeObserver.disconnect(),this.handleScroll&&t.removeEventListener("scroll",this.handleScroll),this.patternAnalyzer&&(this.patternAnalyzer.clear(),this.patternAnalyzer=null,this.isScrollProtectionEnabled=!1),this.timelineEntries.clear(),this.cachedFragmentIds=[],this.sortedFragmentIds=[],this.cachedFragmentStarts.clear(),e.log("WebPubSnapper Unmounted"),!0}};An.moduleName="webpub_snapper";var Dn=An;function In(r,t){let e=t.map(h=>h.toUpperCase()),i=[],o=r.commonAncestorContainer,n=o.nodeType===Node.TEXT_NODE?o.parentNode:o,s=r.startContainer.ownerDocument.createTreeWalker(n,NodeFilter.SHOW_TEXT),a=s.nextNode();for(;a;){if(r.intersectsNode(a)){let h=a.parentNode,l=!1;for(;h;){if(h.nodeType===Node.ELEMENT_NODE&&e.includes(h.tagName.toUpperCase())){l=!0;break}h=h.parentNode}if(!l){let d=r.cloneRange();d.selectNode(a),d.compareBoundaryPoints(Range.START_TO_START,r)<0&&d.setStart(r.startContainer,r.startOffset),d.compareBoundaryPoints(Range.END_TO_END,r)>0&&d.setEnd(r.endContainer,r.endOffset);for(let c of d.getClientRects())i.push({left:c.left,right:c.right,top:c.top,bottom:c.bottom,width:c.width,height:c.height})}}a=s.nextNode()}return i}function Ct(r,t,e=!1,i=0){let o;if(Array.isArray(r))o=r;else{let l=r.getClientRects();l.length||r.commonAncestorContainer.nodeType===Node.ELEMENT_NODE&&(l=r.commonAncestorContainer.getClientRects()),o=[];for(let d of l)o.push({bottom:d.bottom,height:d.height,left:d.left,right:d.right,top:d.top,width:d.width})}if(i)for(let l of o)l.left-=i,l.top-=i,l.right+=i,l.bottom+=i,l.width+=i*2,l.height+=i*2;let n=Hn(o,1,t,e),s=la(n,1),a=Wn(s),h=4;for(let l=a.length-1;l>=0;l--){let d=a[l];if(!(d.width*d.height>h))if(a.length>1)a.splice(l,1);else break}return a}function Hn(r,t,e,i=!1){for(let o=0;o<r.length;o++)for(let n=o+1;n<r.length;n++){let s=r[o],a=r[n];if(s===a)continue;let h=dt(s.top,a.top,t)&&dt(s.bottom,a.bottom,t),l=dt(s.left,a.left,t)&&dt(s.right,a.right,t);if((l&&!e&&!i||h&&!l)&&Bn(s,a,t)){let d=r.filter(m=>m!==s&&m!==a),c=aa(s,a);return d.push(c),Hn(d,t,e,i)}}return r}function aa(r,t){let e=Math.min(r.left,t.left),i=Math.max(r.right,t.right),o=Math.min(r.top,t.top),n=Math.max(r.bottom,t.bottom);return{bottom:n,height:n-o,left:e,right:i,top:o,width:i-e}}function la(r,t){let e=new Set(r);for(let i of r){if(!(i.width>1&&i.height>1)){e.delete(i);continue}for(let o of r)if(i!==o&&e.has(o)&&ha(o,i,t)){e.delete(i);break}}return Array.from(e)}function ha(r,t,e){return ct(r,t.left,t.top,e)&&ct(r,t.right,t.top,e)&&ct(r,t.left,t.bottom,e)&&ct(r,t.right,t.bottom,e)}function ct(r,t,e,i){return(r.left<t||dt(r.left,t,i))&&(r.right>t||dt(r.right,t,i))&&(r.top<e||dt(r.top,e,i))&&(r.bottom>e||dt(r.bottom,e,i))}function Wn(r){for(let t=0;t<r.length;t++)for(let e=t+1;e<r.length;e++){let i=r[t],o=r[e];if(i!==o&&Bn(i,o,-1)){let n=[],s,a=Un(i,o);if(a.length===1)n=a,s=i;else{let l=Un(o,i);a.length<l.length?(n=a,s=i):(n=l,s=o)}let h=r.filter(l=>l!==s);return Array.prototype.push.apply(h,n),Wn(h)}}return r}function Un(r,t){let e=ca(t,r);if(e.height===0||e.width===0)return[r];let i=[];{let o={bottom:r.bottom,height:0,left:r.left,right:e.left,top:r.top,width:0};o.width=o.right-o.left,o.height=o.bottom-o.top,o.height!==0&&o.width!==0&&i.push(o)}{let o={bottom:e.top,height:0,left:e.left,right:e.right,top:r.top,width:0};o.width=o.right-o.left,o.height=o.bottom-o.top,o.height!==0&&o.width!==0&&i.push(o)}{let o={bottom:r.bottom,height:0,left:e.left,right:e.right,top:e.bottom,width:0};o.width=o.right-o.left,o.height=o.bottom-o.top,o.height!==0&&o.width!==0&&i.push(o)}{let o={bottom:r.bottom,height:0,left:e.right,right:r.right,top:r.top,width:0};o.width=o.right-o.left,o.height=o.bottom-o.top,o.height!==0&&o.width!==0&&i.push(o)}return i}function ca(r,t){let e=Math.max(r.left,t.left),i=Math.min(r.right,t.right),o=Math.max(r.top,t.top),n=Math.min(r.bottom,t.bottom);return{bottom:n,height:Math.max(0,n-o),left:e,right:i,top:o,width:Math.max(0,i-e)}}function Bn(r,t,e){return(r.left<t.right||e>=0&&dt(r.left,t.right,e))&&(t.left<r.right||e>=0&&dt(t.left,r.right,e))&&(r.top<t.bottom||e>=0&&dt(r.top,t.bottom,e))&&(t.top<r.bottom||e>=0&&dt(t.top,r.bottom,e))}function dt(r,t,e){return Math.abs(r-t)<=e}var ii=class{constructor(t,e){this.window=t,this.copyHistory=[],this.lastSelectionLength=0,this.lastSelectionTime=0,this.options=e}cleanupOldHistory(t){this.copyHistory=this.copyHistory.filter(e=>t-e.timestamp<1e4),this.copyHistory.length>this.options.historySize&&(this.copyHistory=this.copyHistory.slice(-this.options.historySize))}isSuspiciousPattern(t){return this.copyHistory.length<3?!1:this.copyHistory.filter(e=>t-e.timestamp<2e3).length>=3?!0:this.copyHistory.slice().sort((e,i)=>e.timestamp-i.timestamp).every((e,i,o)=>i===0?!0:e.length>o[i-1].length*1.5)}shouldAllowCopy(t){if(!this.options.enabled)return!0;let e=this.window.getSelection();if(!e)return!0;let i=e.toString().length,o=this.window.document.body.innerText.length,n=Date.now();if(this.cleanupOldHistory(n),i<this.options.minThreshold)return this.copyHistory.push({timestamp:n,length:i,wasBlocked:!1}),!0;let s=n-this.lastSelectionTime<100&&i>this.lastSelectionLength*1.5,a=Math.min(o*this.options.maxSelectionPercent,this.options.absoluteMaxChars),h=this.isSuspiciousPattern(n),l=i>a||s||h;return this.copyHistory.push({timestamp:n,length:i,wasBlocked:l}),l?(t?.preventDefault(),!1):(this.lastSelectionLength=i,this.lastSelectionTime=n,!0)}destroy(){this.lastSelectionLength=0,this.lastSelectionTime=0,this.copyHistory=[],this.options.enabled=!1}};var oi=class{constructor(t=ei){this.options=t,this.events=[],this.selectionStartTime=0,this.lastSelectionTime=0,this.lastSelectionPosition=0,this.selectionPatterns=[],this.lastSelectedText=""}analyze(t){if(!t)return this.clear(),!1;let e=t.toString();if(e.length===0)return this.clear(),!1;if(t.type!=="Range"||!t.rangeCount)return!1;let i=Date.now();if(e.length<=50||e===this.lastSelectedText)return!1;if(this.selectionStartTime===0)return this.selectionStartTime=i,this.lastSelectedText=e,!1;if(i-this.selectionStartTime<500)return!1;i-this.lastSelectionTime>1e3&&(this.lastSelectionTime=i),this.selectionStartTime===0&&(this.selectionStartTime=i),this.lastSelectedText=e;let o=this.analyzeSelectionPattern(t,i);return this.cleanup(i),o}analyzeSelectionPattern(t,e){if(!t.rangeCount)return!1;let i=t.getRangeAt(0),o=i.toString(),n=(e-this.selectionStartTime)/1e3;if(o.length/Math.max(1,n)>this.options.maxSelectionsPerSecond)return!0;let s=i.startOffset,a=Math.abs(s-this.lastSelectionPosition);return this.selectionPatterns.push(a),this.selectionPatterns.length>this.options.historySize&&(this.selectionPatterns.shift(),this.calculateVariance(this.selectionPatterns)<this.options.minVariance)?!0:(this.lastSelectionPosition=s,!1)}calculateVariance(t){if(t.length===0)return 0;let e=t.reduce((i,o)=>i+o,0)/t.length;return t.reduce((i,o)=>i+Math.pow(o-e,2),0)/t.length}cleanup(t){this.events=this.events.filter(e=>t-e.timestamp<=1e3)}clear(){this.events=[],this.selectionStartTime=0,this.lastSelectionTime=0,this.lastSelectionPosition=0,this.selectionPatterns=[],this.lastSelectedText=""}};var Ht=class{match(t,e){for(let i of e)if(this.matchesCombo(t,i))return!0;return!1}matchesCombo(t,e){return t.keyCode===e.keyCode&&this.matchesModifier(t.ctrlKey,e.ctrl)&&this.matchesModifier(t.shiftKey,e.shift)&&this.matchesModifier(t.altKey,e.alt)&&this.matchesModifier(t.metaKey,e.meta)}matchesModifier(t,e){return e===void 0?!t:t===e}createKeyHandler(t,e){return i=>{this.match(i,t)&&(i.preventDefault(),i.stopPropagation(),e(i))}}createActivityEvent(t,e,i,o){let n,s;if(o){let a=o.getSelection(),h=a?.toString()||"",l=(h&&a?.rangeCount?a.getRangeAt(0)?.getClientRects():null)?.[0];l&&h&&(n={text:h,x:l.x,y:l.y,width:l.width,height:l.height});let d=o.document.activeElement;d&&d!==o.document.body&&(s=Ee(d)?.outerHTML)}return{type:e,timestamp:Date.now(),key:t.key,code:t.code,keyCode:t.keyCode,ctrlKey:t.ctrlKey,altKey:t.altKey,shiftKey:t.shiftKey,metaKey:t.metaKey,targetFrameSrc:i,selectedText:n,interactiveElement:s}}createKeyboardHandlers(t,e,i,o){let n=[];return e.forEach(s=>{n.push(...s.keyCombos.map(a=>({...a,handler:h=>{let l=s.type,d=this.createActivityEvent(h,l,t,o);i(d)}})))}),n}createUnifiedHandler(t,e,i,o){let n=this.createKeyboardHandlers(t,e,i,o);return s=>{if(s.isTrusted){for(let a of n)if(this.match(s,[a])){let h=a.suppressOnInteractiveElement;if(h){let l=(o?.document??document).activeElement;if(Array.isArray(h)?h.some(d=>l?.matches(d)):go(l))return}s.preventDefault(),s.stopPropagation(),a.handler(s);return}}}}};var jn=class ce extends Q{constructor(){super(...arguments),this.configApplied=!1,this.cleanupCallbacks=[],this.pointerMoved=!1,this.isContextMenuEnabled=!1,this.isDragAndDropEnabled=!1,this.isSelectionMonitoringEnabled=!1,this.isBulkCopyProtectionEnabled=!1,this.selectionAnalyzer=null,this.currentSelection=null,this.bulkCopyProtector=null,this.keyManager=new Ht,this.keyDownHandler=null,this.preventBulkCopy=t=>{if(!this.isBulkCopyProtectionEnabled||!this.bulkCopyProtector)return!0;if(!this.bulkCopyProtector.shouldAllowCopy(t)){t.preventDefault();let e=this.wnd.getSelection(),i=e?.toString()||"",o=(i?e?.getRangeAt(0)?.getClientRects():null)?.[0],n={type:"bulk_copy",timestamp:Date.now(),clipboardTypes:t.clipboardData?.types?[...t.clipboardData.types]:[],selectedText:o?{text:i,x:o.x,y:o.y,width:o.width,height:o.height}:void 0,selectionLength:i.length,targetFrameSrc:this.wnd.location.href};return this.comms?.send("content_protection",n),!1}return!0},this.handleSelection=t=>{if(!this.isSelectionMonitoringEnabled||!this.wnd||!this.selectionAnalyzer)return;let e=this.wnd.getSelection();if(e){if(this.currentSelection=e.toString(),this.selectionAnalyzer.analyze(e)&&this.currentSelection){let i=this.wnd.getSelection(),o=i?.toString()||"",n=(o&&i?.rangeCount?i.getRangeAt(0)?.getClientRects():null)?.[0],s={type:"suspicious_selection",timestamp:Date.now(),selectionLength:o.length,selectedText:{text:o,x:n?.x??0,y:n?.y??0,width:n?.width??0,height:n?.height??0},eventType:t?.type||"selectionchange",targetFrameSrc:this.wnd.location.href};this.comms?.send("content_protection",s)}}else this.currentSelection=null},this.onDragOver=t=>{this.isDragAndDropEnabled&&(t.preventDefault(),t.stopPropagation())},this.onDragStart=t=>{if(this.isDragAndDropEnabled){t.preventDefault();let e={type:"drag_detected",timestamp:Date.now(),dataTransferTypes:t.dataTransfer?.types?[...t.dataTransfer.types]:[],targetFrameSrc:this.wnd.location.href};return this.comms?.send("content_protection",e),!1}else return!0},this.onDrop=t=>{if(this.isDragAndDropEnabled){t.preventDefault();let e=t.dataTransfer,i={type:"drop_detected",timestamp:Date.now(),dataTransferTypes:e?.types?[...e.types]:[],fileCount:e?.files?.length||0,targetFrameSrc:this.wnd.location.href};return this.comms?.send("content_protection",i),!1}else return!0},this.onContext=t=>{if(this.isContextMenuEnabled){t.preventDefault();let e=this.wnd.getSelection(),i=e?.toString()||"",o=(i&&e?.rangeCount?e.getRangeAt(0)?.getClientRects():null)?.[0],n={timestamp:Date.now(),clientX:t.clientX,clientY:t.clientY,...o&&{selectedText:{text:i,x:o.x,y:o.y,width:o.width,height:o.height}},targetFrameSrc:this.wnd.location.href};this.comms?.send("context_menu",n)}},this.onPointerUp=this.onPointUp.bind(this),this.onPointerMove=this.onPointMove.bind(this),this.onPointerDown=this.onPointDown.bind(this),this.onClicker=this.onClick.bind(this)}addContextMenuPrevention(){this.isContextMenuEnabled||!this.wnd||(this.wnd.document.addEventListener("contextmenu",this.onContext),this.isContextMenuEnabled=!0)}removeContextMenuPrevention(){!this.isContextMenuEnabled||!this.wnd||(this.wnd.document.removeEventListener("contextmenu",this.onContext),this.isContextMenuEnabled=!1)}addDragAndDropPrevention(){this.isDragAndDropEnabled||!this.wnd||(this.wnd.document.addEventListener("dragstart",this.onDragStart),this.wnd.document.addEventListener("dragover",this.onDragOver),this.wnd.document.addEventListener("drop",this.onDrop),this.isDragAndDropEnabled=!0)}removeDragAndDropPrevention(){!this.isDragAndDropEnabled||!this.wnd||(this.wnd.document.removeEventListener("dragstart",this.onDragStart),this.wnd.document.removeEventListener("dragover",this.onDragOver),this.wnd.document.removeEventListener("drop",this.onDrop),this.isDragAndDropEnabled=!1)}enableKeyboardPeripherals(t=[]){this.disableKeyboardPeripherals();let e=i=>{this.comms?.send("keyboard_peripherals",i)};this.keyDownHandler=this.keyManager.createUnifiedHandler(this.wnd.location.href,t,e,this.wnd),this.wnd&&this.wnd.document.addEventListener("keydown",this.keyDownHandler,{capture:!0})}disableKeyboardPeripherals(){this.wnd&&this.keyDownHandler&&(this.wnd.document.removeEventListener("keydown",this.keyDownHandler,{capture:!0}),this.keyDownHandler=null)}addBulkCopyProtection(t={}){if(this.isBulkCopyProtectionEnabled||!this.wnd)return;let e=kn,i=t?{...e,...t}:e;this.bulkCopyProtector=new ii(this.wnd,i),this.wnd.document.addEventListener("copy",this.preventBulkCopy,!0),this.wnd.document.addEventListener("cut",this.preventBulkCopy,!0),this.isBulkCopyProtectionEnabled=!0}removeBulkCopyProtection(){!this.isBulkCopyProtectionEnabled||!this.wnd||(this.wnd.document.removeEventListener("copy",this.preventBulkCopy,!0),this.wnd.document.removeEventListener("cut",this.preventBulkCopy,!0),this.bulkCopyProtector?.destroy(),this.bulkCopyProtector=null,this.isBulkCopyProtectionEnabled=!1)}addSelectionMonitoring(t){if(this.isSelectionMonitoringEnabled||!this.wnd)return;let e=t||ei;this.selectionAnalyzer=new oi(e),this.wnd.document.addEventListener("selectionchange",this.handleSelection),this.isSelectionMonitoringEnabled=!0}removeSelectionMonitoring(){!this.isSelectionMonitoringEnabled||!this.wnd||(this.wnd.document.removeEventListener("selectionchange",this.handleSelection),this.selectionAnalyzer?.clear(),this.selectionAnalyzer=null,this.isSelectionMonitoringEnabled=!1)}onPointUp(t){let e=this.wnd.getSelection();if(e&&e.toString()?.length>0){let o=e.getRangeAt(0)?.getClientRects();if(!o||o.length===0)return;let n=o[0],s={text:e.toString(),x:n.x,y:n.y,width:n.width,height:n.height,targetFrameSrc:this.wnd?.location?.href};this.comms.send("text_selected",s)}if(this.pointerMoved){this.pointerMoved=!1;return}if(!e?.isCollapsed||!t.isPrimary)return;let i=this.wnd.devicePixelRatio;t.preventDefault(),this.comms.send(t.pointerType==="touch"?"tap":"click",{defaultPrevented:t.defaultPrevented,x:t.clientX*i,y:t.clientY*i,targetFrameSrc:this.wnd.location.href,targetElement:t.target.outerHTML,interactiveElement:Ee(t.target)?.outerHTML,cssSelector:this.wnd._readium_cssSelectorGenerator.getCssSelector(t.target),onContent:this.isOnContent(t.target,t.clientX,t.clientY)}),this.pointerMoved=!1}onPointMove(t){if(t.movementY!==void 0&&t.movementX!==void 0){(Math.abs(t.movementX)>1||Math.abs(t.movementY)>1)&&(this.pointerMoved=!0);return}this.pointerMoved=!0}onPointDown(){this.pointerMoved=!1}onClick(t){if(t.preventDefault(),!t.isTrusted){let e=new PointerEvent("pointerup",{isPrimary:!0,pointerType:"mouse",clientX:t.clientX,clientY:t.clientY});Object.defineProperty(e,"target",{writable:!1,value:t.target}),Object.defineProperty(e,"defaultPrevented",{writable:!1,value:t.defaultPrevented}),this.onPointUp(e)}}isOnContent(t,e,i){if(t.closest("img, svg, video, audio, canvas, object"))return!0;let o=this.wnd.document.createRange();for(let n of Array.from(t.childNodes))if(!(n.nodeType!==Node.TEXT_NODE||!n.textContent?.trim())&&(o.selectNodeContents(n),Array.from(o.getClientRects()).some(s=>ct(s,e,i,0))))return!0;return!1}pointedPiece({cssSelector:t,x:e,y:i,pieces:o}){let n=this.wnd.document.querySelector(t);if(!n)return-1;let s=this.wnd.document.createRange();s.selectNode(n);let a=this.wnd.devicePixelRatio,h=-1;for(let l=0;l<o.length;l++){let[d,c]=o[l],m=ae(this.wnd.document,new z({href:this.wnd.location.href,type:"text/html",text:at.deserialize(d),locations:c?new G({otherLocations:new Map([["cssSelector",c]])}):void 0}));if(!(!m||!m.intersectsNode(n))){if(Ct(m,!1).some(p=>ct(p,e/a,i/a,0)))return l;h===-1&&m.compareBoundaryPoints(Range.START_TO_START,s)<=0&&m.compareBoundaryPoints(Range.END_TO_END,s)>=0&&(h=l)}}return h}registerPointedPiece(){this.comms.register("pointed_piece",ce.moduleName,(t,e)=>{let i=t;this.comms.send("pointed_piece",{id:i.id,index:this.pointedPiece(i)}),e(!0)})}registerProtectionHandlers(){this.comms?.register("peripherals_protection",ce.moduleName,(t,e)=>{let i=t;if(!this.configApplied){if(this.configApplied=!0,i.monitorSelection){let o=typeof i.monitorSelection=="boolean"?void 0:i.monitorSelection;this.addSelectionMonitoring(o),this.comms?.log("Selection monitoring enabled")}typeof i.protectCopy=="object"?(this.addBulkCopyProtection({enabled:!0,...i.protectCopy}),this.comms?.log("Copy protection enabled (limited)")):i.protectCopy===!0&&(this.addBulkCopyProtection({enabled:!0,maxSelectionPercent:0,minThreshold:0,absoluteMaxChars:0}),this.comms?.log("Copy protection enabled")),i.disableContextMenu&&(this.addContextMenuPrevention(),this.comms?.log("Context menu protection enabled")),i.disableDragAndDrop&&(this.addDragAndDropPrevention(),this.comms?.log("Drag and drop protection enabled"))}e(!0)}),this.comms?.register("unfocus",ce.moduleName,(t,e)=>{this.disableKeyboardPeripherals(),e(!0)}),this.comms?.register("keyboard_peripherals",ce.moduleName,(t,e)=>{let i=t;i&&i.length>0&&(this.enableKeyboardPeripherals(i),this.comms?.log(`Keyboard peripherals enabled: ${i.map(o=>o.type).join(", ")}`)),e(!0)})}mount(t,e){return this.wnd=t,this.comms=e,this.registerProtectionHandlers(),this.registerPointedPiece(),t.document.addEventListener("pointerdown",this.onPointerDown),t.document.addEventListener("pointerup",this.onPointerUp),t.document.addEventListener("pointermove",this.onPointerMove),t.document.addEventListener("click",this.onClicker),e.log("Peripherals Mounted"),!0}unmount(t,e){return this.removeBulkCopyProtection(),this.removeSelectionMonitoring(),this.removeContextMenuPrevention(),this.removeDragAndDropPrevention(),this.disableKeyboardPeripherals(),this.cleanupCallbacks.forEach(i=>i()),this.cleanupCallbacks=[],t.document.removeEventListener("pointerdown",this.onPointerDown),t.document.removeEventListener("pointerup",this.onPointerUp),t.document.removeEventListener("pointermove",this.onPointerMove),t.document.removeEventListener("click",this.onClicker),e.unregisterAll(ce.moduleName),this.configApplied=!1,e.log("Peripherals Unmounted"),!0}};jn.moduleName="peripherals";var Gn=jn;var Vn=class fo extends Q{constructor(){super(...arguments),this.mediaPlayingCount=0,this.allAnimations=new Set}wndOnErr(t){this.comms?.send("error",{message:t.message,filename:t.filename,lineno:t.lineno,colno:t.colno})}unblock(t){for(t._readium_blockEvents=!1;t._readium_blockedEvents?.length>0;){let e=t._readium_blockedEvents.shift();switch(e[0]){case 0:Reflect.apply(e[1],e[2],e[3]);break;case 1:let i=e[1],o=e[2];t.removeEventListener(i.type,t._readium_eventBlocker,!0);let n=new Event(i.type,{bubbles:i.bubbles,cancelable:i.cancelable});o?o.dispatchEvent(n):t.dispatchEvent(n);break}}}onMediaPlayEvent(){this.mediaPlayingCount++,this.comms?.send("media_play",this.mediaPlayingCount)}onMediaPauseEvent(){this.mediaPlayingCount>0&&this.mediaPlayingCount--,this.comms?.send("media_pause",this.mediaPlayingCount)}pauseAllMedia(t){let e=t.document.querySelectorAll("audio,video");for(let i=0;i<e.length;i++)e[i].pause()}mount(t,e){this.comms=e,t.addEventListener("error",this.wndOnErr,!1),Reflect.defineProperty(t.navigator,"epubReadingSystem",{value:{name:"readium-ts-toolkit",version:"2.9.2",hasFeature:(o,n="")=>{switch(o){case"dom-manipulation":return!0;case"layout-changes":return!0;case"touch-events":return!0;case"mouse-events":return!0;case"keyboard-events":return!0;case"spine-scripting":return!0;case"embedded-web-content":return!0;default:return!1}}},writable:!1}),"getAnimations"in t.document&&t.document.getAnimations().forEach(o=>{o.cancel(),this.allAnimations.add(o)}),e.register("activate",fo.moduleName,(o,n)=>{this.allAnimations.forEach(s=>{s.cancel(),s.play()}),n(!0)}),e.register("unfocus",fo.moduleName,(o,n)=>{this.pauseAllMedia(t),this.allAnimations.forEach(s=>s.pause()),n(!0)});let i=t.document.querySelectorAll("audio,video");for(let o=0;o<i.length;o++){let n=i[o];n.addEventListener("play",this.onMediaPlayEvent,{passive:!0}),n.addEventListener("pause",this.onMediaPauseEvent,{passive:!0})}return e.log("Setup Mounted"),!0}unmount(t,e){return t.removeEventListener("error",this.wndOnErr),t.removeEventListener("play",this.onMediaPlayEvent),t.removeEventListener("pause",this.onMediaPauseEvent),this.allAnimations.forEach(i=>i.cancel()),this.allAnimations.clear(),e.log("Setup Unmounted"),!0}};Vn.moduleName="setup";var ni=Vn;var Xn="readium-viewport",$n=class Wt extends ni{onViewportWidthChanged(t){let e=t.target;Rt(e,"--RS__viewportWidth",`${e.innerWidth}px`)}mount(t,e){if(!super.mount(t,e))return!1;let i=t.document.createElement("meta");return i.dataset.readium="true",i.setAttribute("name","viewport"),i.setAttribute("id",Xn),i.setAttribute("content","width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, shrink-to-fit=no"),t.document.head.appendChild(i),t.addEventListener("orientationchange",this.onViewportWidthChanged),t.addEventListener("resize",this.onViewportWidthChanged),this.onViewportWidthChanged({target:t}),e.register("get_properties",Wt.moduleName,(o,n)=>{Ce(t),n(!0)}),e.register("update_properties",Wt.moduleName,(o,n)=>{o["--RS__viewportWidth"]=`${t.innerWidth}px`,ti(t,o),n(!0)}),e.register("set_property",Wt.moduleName,(o,n)=>{let s=o;Rt(t,s[0],s[1]),n(!0)}),e.register("remove_property",Wt.moduleName,(o,n)=>{It(t,o),n(!0)}),e.register("activate",Wt.moduleName,(o,n)=>{this.unblock(t),n(!0)}),e.log("ReflowableSetup Mounted"),!0}unmount(t,e){return e.unregisterAll(Wt.moduleName),t.document.head.querySelector(`#${Xn}`)?.remove(),t.removeEventListener("orientationchange",this.onViewportWidthChanged),e.log("ReflowableSetup Unmounted"),super.unmount(t,e)}};$n.moduleName="reflowable_setup";var Kn=$n;var Yn="readium-fixed-style",qn=class Et extends ni{mount(t,e){if(!super.mount(t,e))return!1;t.navigator.epubReadingSystem&&(t.navigator.epubReadingSystem.layoutStyle="paginated");let i=t.document.createElement("style");return i.id=Yn,i.dataset.readium="true",i.textContent=`
        html, body {
            text-size-adjust: none;
            -ms-text-size-adjust: none;
            -webkit-text-size-adjust: none;
            -moz-text-size-adjust: none;

            /* Fight Safari pinches */
            touch-action: none !important;
            min-height: 100%;

            /*cursor: var() TODO*/
        }`,t.document.head.appendChild(i),e.register("set_property",Et.moduleName,(o,n)=>{let s=o;Rt(t,s[0],s[1]),n(!0)}),e.register("remove_property",Et.moduleName,(o,n)=>{It(t,o),n(!0)}),e.register("first_visible_locator",Et.moduleName,(o,n)=>n(!1)),e.register("unfocus",Et.moduleName,(o,n)=>{F(t),n(!0)}),e.register(["focus","go_next","go_prev","go_id","go_end","go_start","go_text","go_progression"],Et.moduleName,(o,n)=>n(!0)),e.register("activate",Et.moduleName,(o,n)=>{this.unblock(t),n(!0)}),e.log("FixedSetup Mounted"),!0}unmount(t,e){return e.unregisterAll(Et.moduleName),t.document.getElementById(Yn)?.remove(),e.log("FixedSetup Unmounted"),super.unmount(t,e)}};qn.moduleName="fixed_setup";var Jn=qn;var Zn=["div","span","p","br","hr","b","i","em","strong","s","u","mark","small","sub","sup","abbr","cite","code","data","dfn","kbd","q","samp","time","var","blockquote","pre","svg","g","path","circle","ellipse","rect","line","polygon","polyline","text","tspan","defs","use"],da=/^on/i,ma=new Set(["href","src","action","formaction","xlink:href"]),ua=/^\s*(javascript|data):/i;function Qn(r,t){let e=r.document.createElement("div");if("Sanitizer"in r&&typeof e.setHTML=="function")try{let o=new r.Sanitizer({allowElements:Zn});return e.setHTML(t,{sanitizer:o}),e.firstElementChild}catch{}let i=r.document.implementation.createHTMLDocument("");for(i.body.innerHTML=t,pa(i.body,new Set(Zn));i.body.firstChild;)e.appendChild(r.document.adoptNode(i.body.firstChild));return e.firstElementChild}function pa(r,t){let e=Array.from(r.querySelectorAll("*")).reverse();for(let i of e){if(!t.has(i.localName)){i.replaceWith(...Array.from(i.childNodes));continue}for(let{name:o,value:n}of Array.from(i.attributes))(da.test(o)||ma.has(o)&&ua.test(n))&&i.removeAttribute(o)}}function So(r){switch(r){case R.Mask:return"rgba(255, 255, 255, 0.5)";case R.Highlight:case R.HighlightUnderline:return"#FFFF00";default:return"#FF0000"}}var R={Highlight:"highlight",HighlightUnderline:"highlightUnderline",Underline:"underline",Strikethrough:"strikethrough",Outline:"outline",TextColor:"textColor",Mask:"mask",Template:"template"},ir=(r=>(r.Wrap="wrap",r.Viewport="viewport",r.Bounds="bounds",r.Page="page",r))(ir||{}),or=(r=>(r.Boxes="boxes",r.Bounds="bounds",r))(or||{}),ga=()=>"Highlight"in window,tr=["IMG","IMAGE","AUDIO","VIDEO","SVG"],fa=new Set(["SUP","SUB","SMALL","CODE"]),Pe=2,ri=1,er=ri+Pe/2,yo=class r{constructor(t,e,i,o){this.wnd=t,this.comms=e,this.id=i,this.name=o,this.items=[],this.lastItemId=0,this.container=void 0,this._activatable=!1,this._hoverable=!1,this.hoveredItem=void 0,this.experimentalHighlights=!1,this._tintSubKeys=new Map,this._subKeyCounter=0,this.pendingWebkitReflow=null,this.maskSvg=void 0,this.shadowHost=void 0,this.shadowRoot=void 0,this.currentRender=0,this.pendingRender=new Set,this.maskFrame=0,ga()&&(this.experimentalHighlights=!0,this.notTextFlag=new Map),this.activationHandler=this.handleActivation.bind(this),this.wnd.document.addEventListener("pointerup",this.activationHandler),this.hoverHandler=this.handleHover.bind(this),this.wnd.document.addEventListener("pointermove",this.hoverHandler)}get activatable(){return this._activatable}set activatable(t){this._activatable=t}get hoverable(){return this._hoverable}get hasOverlayItems(){return!this.experimentalHighlights||(this.notTextFlag?.size??0)>0}set hoverable(t){if(this._hoverable=t,!t&&this.hoveredItem){let e=this.hoveredItem.range.getBoundingClientRect(),i=this.wnd.devicePixelRatio;this.comms.send("decoration_pointer_leave",{decorationId:this.hoveredItem.decoration.id,group:this.name,rect:{top:e.top*i,left:e.left*i,width:e.width*i,height:e.height*i}}),this.hoveredItem=void 0}}add(t){let e=`${this.id}-${this.lastItemId++}`,i=ae(this.wnd.document,t.locator);if(!i){this.comms.log("Can't locate DOM range for decoration",t);return}let o=i.commonAncestorContainer;if(o.nodeType!==Node.TEXT_NODE&&this.experimentalHighlights&&(tr.includes(o.nodeName.toUpperCase())&&this.notTextFlag?.set(e,!0),i.cloneContents().querySelector(tr.join(", ").toLowerCase())&&this.notTextFlag?.set(e,!0),(o.textContent?.trim()||"").length===0&&this.notTextFlag?.set(e,!0)),this.experimentalHighlights&&!this.notTextFlag?.has(e)){let s=a=>{for(;a&&a.nodeType===Node.ELEMENT_NODE;){if(a.namespaceURI?.includes("svg"))return!0;a=a.parentNode}return!1};(s(i.startContainer)||s(i.endContainer))&&this.notTextFlag?.set(e,!0)}if(this.experimentalHighlights){let{type:s}=t.style,{layout:a,width:h,expand:l}=t.style;s!==R.TextColor&&(s===R.Outline||s===R.Template||s===R.Mask||a!==void 0&&a!=="boxes"||h!==void 0&&h!=="wrap"||l)&&this.notTextFlag?.set(e,!0)}let n={decoration:t,id:e,range:i,hitRects:[],clickableElements:void 0,container:void 0};this.items.push(n),this.layout(n),n.hitRects=this.clientRectsToDocCoords(Ct(n.range,!1,!1,(n.decoration.style.expand??0)+this.hitGap())),this.renderLayout([n])}remove(t){let e=this.items.findIndex(n=>n.decoration.id===t);if(e<0)return;let i=this.items[e],o=i.decoration.style?.type===R.Mask;if(this.items.splice(e,1),i.clickableElements=void 0,i.container&&(i.container.remove(),i.container=void 0),this.experimentalHighlights&&!this.notTextFlag?.has(i.id)&&i.highlightSubKey){let n=this.wnd.CSS.highlights;n.get(i.highlightSubKey)?.delete(i.range),this.items.some(a=>a.highlightSubKey===i.highlightSubKey)||n.delete(i.highlightSubKey);let s=this.wnd.document.getElementById(`${this.id}-style`);s&&this._rebuildHighlightStylesheet(s),this.scheduleWebkitRepaintFix([i])}this.notTextFlag?.delete(i.id),this.hoveredItem===i&&(this.hoveredItem=void 0),o&&this.scheduleMaskUpdate()}update(t){this.remove(t.id),this.add(t)}clear(){this.experimentalHighlights&&this.scheduleWebkitRepaintFix(this.items),this.clearContainer(),this.items.length=0,this.notTextFlag?.clear(),this.hoveredItem=void 0,this.maskSvg?this.scheduleMaskUpdate():this.removeShadowHost()}destroy(){this.clear(),this.wnd.cancelAnimationFrame(this.maskFrame),this.maskFrame=0,this.maskSvg?.remove(),this.maskSvg=void 0,this.removeShadowHost(),this.wnd.document.removeEventListener("pointerup",this.activationHandler),this.wnd.document.removeEventListener("pointermove",this.hoverHandler)}isWebkitRepaintProne(t){let e=this.wnd.getComputedStyle(t);if(fa.has(t.tagName.toUpperCase())||e.display==="inline-block"||e.verticalAlign!=="baseline")return!0;let i=t.parentElement?this.wnd.getComputedStyle(t.parentElement):null;return!!i&&parseFloat(e.fontSize)!==parseFloat(i.fontSize)}findWebkitRepaintProneElement(t){let e=t.startContainer.nodeType===Node.ELEMENT_NODE?t.startContainer:t.startContainer.parentElement;for(;e;){if(this.isWebkitRepaintProne(e))return e;if(this.wnd.getComputedStyle(e).display!=="inline")break;e=e.parentElement}let i=t.commonAncestorContainer,o=i.nodeType===Node.ELEMENT_NODE?i:i.parentElement;if(!o)return null;let n=this.wnd.document.createTreeWalker(o,NodeFilter.SHOW_ELEMENT,s=>t.intersectsNode(s)?NodeFilter.FILTER_ACCEPT:NodeFilter.FILTER_REJECT);for(let s=n.nextNode();s;s=n.nextNode())if(this.isWebkitRepaintProne(s))return s;return null}forceWebkitBlockReflow(t){let e=t;for(;e;){let s=this.wnd.getComputedStyle(e).display;if(s!=="inline"&&s!=="inline-block")break;e=e.parentElement}if(!e||!e.isConnected)return;let i=e,o=i.style.getPropertyValue("transform"),n=i.style.getPropertyPriority("transform");i.style.setProperty("transform","translateZ(0.001px)","important"),i.offsetHeight,i.style.setProperty("transform","translateZ(0px)","important"),i.offsetHeight,o?i.style.setProperty("transform",o,n):i.style.removeProperty("transform")}scheduleWebkitRepaintFix(t){if(j.UA.WebKit)for(let e of t){if(this.notTextFlag?.has(e.id)||!e.highlightSubKey)continue;let i=this.findWebkitRepaintProneElement(e.range);i&&this.queueWebkitReflow(i)}}queueWebkitReflow(t){this.pendingWebkitReflow||(this.pendingWebkitReflow=new Set,this.wnd.requestAnimationFrame(()=>{let e=this.pendingWebkitReflow;this.pendingWebkitReflow=null,e?.forEach(i=>this.forceWebkitBlockReflow(i))})),this.pendingWebkitReflow.add(t)}clientRectsToDocCoords(t,e=Ut(this.wnd)){let i=e.xDocOffset,o=e.yDocOffset;return i===0&&o===0?t:t.map(n=>({left:n.left+i,top:n.top+o,right:n.right+i,bottom:n.bottom+o,width:n.width,height:n.height}))}pointerToDocCoords(t){let e=Ut(this.wnd);return{docX:t.clientX+e.xDocOffset,docY:t.clientY+e.yDocOffset}}effectiveZoom(){if(!j.UA.Blink)return 1;let t=parseFloat(this.wnd.getComputedStyle(this.wnd.document.documentElement).zoom),e=parseFloat(this.wnd.getComputedStyle(this.wnd.document.body).zoom);return(t||1)*(e||1)}hitGap(){return 2*this.effectiveZoom()}handleActivation(t){if(!this._activatable)return;let{docX:e,docY:i}=this.pointerToDocCoords(t),o=this.wnd.devicePixelRatio;for(let n of this.items){let s;if(n.decoration.style.type===R.Template)for(let a of n.clickableElements??[]){let h=a.getBoundingClientRect();if(ct(h,t.clientX,t.clientY,0)){s=h;break}}else for(let a of n.hitRects)if(ct(a,e,i,0)){s=n.range.getBoundingClientRect();break}if(s){this.comms.send("decoration_activated",{decorationId:n.decoration.id,group:this.name,rect:{top:s.top*o,left:s.left*o,width:s.width*o,height:s.height*o},point:{x:t.clientX*o,y:t.clientY*o}});return}}}handleHover(t){if(!this._hoverable)return;let{docX:e,docY:i}=this.pointerToDocCoords(t),o=this.wnd.devicePixelRatio,n,s;for(let a of this.items){if(a.decoration.style.type===R.Template)for(let h of a.clickableElements??[]){let l=h.getBoundingClientRect();if(ct(l,t.clientX,t.clientY,0)){n=a,s=l;break}}else for(let h of a.hitRects)if(ct(h,e,i,0)){n=a,s=a.range.getBoundingClientRect();break}if(n)break}if(n!==this.hoveredItem){if(this.hoveredItem){let a=this.hoveredItem.range.commonAncestorContainer.isConnected?this.hoveredItem.range.getBoundingClientRect():null;this.comms.send("decoration_pointer_leave",{decorationId:this.hoveredItem.decoration.id,group:this.name,rect:a?{top:a.top*o,left:a.left*o,width:a.width*o,height:a.height*o}:void 0,point:{x:t.clientX*o,y:t.clientY*o}})}this.hoveredItem=n,n&&s&&this.comms.send("decoration_pointer_enter",{decorationId:n.decoration.id,group:this.name,rect:{top:s.top*o,left:s.left*o,width:s.width*o,height:s.height*o},point:{x:t.clientX*o,y:t.clientY*o}})}}requestLayout(){this.wnd.cancelAnimationFrame(this.currentRender),this.clearContainer(),this.wnd.document.fonts.ready.then(()=>{this.currentRender=this.wnd.requestAnimationFrame(()=>{this.items.forEach(t=>{this.layout(t),t.hitRects=this.clientRectsToDocCoords(Ct(t.range,!1,!1,(t.decoration.style.expand??0)+this.hitGap()))}),this.renderLayout(this.items)})})}repositionOverlays(){let t=[],e=Ut(this.wnd);this.items.forEach(i=>{if(!(this.experimentalHighlights&&!this.notTextFlag?.has(i.id))){if(i.decoration.style?.type===R.Mask){this.scheduleMaskUpdate();return}this.repositionItem(i,e)||(i.container?.remove(),this.layout(i),t.push(i)),i.hitRects=this.clientRectsToDocCoords(Ct(i.range,!1,!1,(i.decoration.style.expand??0)+this.hitGap()),e)}}),t.length&&this.renderLayout(t)}experimentalLayout(t){let e=this.requireContainer(!0),i=this.wnd.CSS.highlights,o=t.decoration.style,n=o.type??R.Highlight,s=o.tint??So(n),a=o.width,h=o.layout,l=this._getSubKey(n,s);t.highlightSubKey&&(i.get(t.highlightSubKey)?.delete(t.range),t.highlightSubKey!==l&&!this.items.some(u=>u!==t&&u.highlightSubKey===t.highlightSubKey)&&i.delete(t.highlightSubKey)),t.highlightSubKey=l;let d;i.has(l)?d=i.get(l):(d=new this.wnd.Highlight,i.set(l,d));let c=(u,f)=>this.wnd.document.caretPositionFromPoint?.(u,f)??null;if(n===R.TextColor&&(h==="bounds"||a==="bounds"||a==="page")){let u=Ut(this.wnd);if(u.isVertical)console.warn("Vertical writing detected: caretPositionFromPoint has known bugs, falling back to original range"),d.add(t.range);else{let f=t.range.getBoundingClientRect(),S,b;a==="page"?(S=Math.floor(u.inlineStart(f)/u.pageInlineSize)*u.pageInlineSize,b=u.pageInlineSize):(S=u.inlineStart(f),b=u.inlineSize(f));let C=c(S,u.blockStart(f)+1),v=c(S+b,u.blockStart(f)+u.blockSize(f)-1);if(C&&v){let w=this.wnd.document.createRange();w.setStart(C.offsetNode,C.offset),w.setEnd(v.offsetNode,v.offset),d.add(w),t.range=w}else d.add(t.range)}}else d.add(t.range);let m=this.getBackgroundColor(),p=o.enforceContrast!==!1?St(s,m):s,g;switch(n){case R.Underline:g=`::highlight(${l}) {
                    text-decoration: underline;
                    text-decoration-color: ${p};
                    text-decoration-thickness: 0.1em;
                }`;break;case R.Strikethrough:g=`::highlight(${l}) {
                    text-decoration: line-through;
                    text-decoration-color: ${p};
                    text-decoration-thickness: 0.1em;
                }`;break;case R.Outline:g=`::highlight(${l}) {
                    outline: 2px solid ${p};
                    outline-offset: 1px;
                }`;break;case R.TextColor:g=`::highlight(${l}) {
                    color: ${p};
                }`;break;case R.HighlightUnderline:{let{r:u,g:f,b:S}=ft(p),b=`rgba(${u}, ${f}, ${S}, 0.3)`;g=`::highlight(${l}) {
                    color: ${Be(b,m)};
                    background-color: ${b};
                    text-decoration: underline;
                    text-decoration-color: ${p};
                    text-decoration-thickness: 0.1em;
                }`;break}case R.Highlight:default:g=`::highlight(${l}) {
                    color: ${Be(p,m)};
                    background-color: ${p};
                }`}t.highlightCSS=g,this._rebuildHighlightStylesheet(e)}_getSubKey(t,e){let i=`${t}::${e}`,o=this._tintSubKeys.get(i);return o||(o=`${this.id}--${this._subKeyCounter++}`,this._tintSubKeys.set(i,o)),o}_rebuildHighlightStylesheet(t){let e=new Set,i=[];for(let o of this.items)o.highlightSubKey&&o.highlightCSS&&!e.has(o.highlightSubKey)&&(e.add(o.highlightSubKey),i.push(o.highlightCSS));t.innerHTML=i.join(`
`)}positionElement(t,e,i,o,n,s,a=0){let[h,l,d,c]=this.overlayBox(t,e,n,s,a);e.applyPosition(o,h,l,d,c,i)}overlayBox(t,e,i,o,n=0){let s=t.decoration?.style?.width,a=i;switch(s){case"viewport":return[Math.floor(e.inlineStart(a)/e.viewportInlineSize)*e.viewportInlineSize+e.inlineScrollOffset+n,e.blockStart(a)+e.blockScrollOffset,e.viewportInlineSize-2*n,e.blockSize(a)];case"page":return[Math.floor(e.inlineStart(a)/e.pageInlineSize)*e.pageInlineSize+e.inlineScrollOffset+n,e.blockStart(a)+e.blockScrollOffset,e.pageInlineSize-2*n,e.blockSize(a)];case"bounds":return[e.inlineStart(o)+e.inlineScrollOffset,e.blockStart(a)+e.blockScrollOffset,e.inlineSize(o),e.blockSize(a)];default:return[e.inlineStart(a)+e.inlineScrollOffset,e.blockStart(a)+e.blockScrollOffset,e.inlineSize(a),e.blockSize(a)]}}static isShapeOutline(t){return t.type===R.Outline&&t.layout!=="bounds"}outlinePath(t,e,i,o,n,s,a){let h=e.isVertical&&!e.isVertLR?-1:1,l=o.map(m=>{let[p,g,u,f]=this.overlayBox(t,e,m,n,s).map(C=>C*i),S=p+u,b=g+f;return h===1?{i0:p,i1:S,b0:g,b1:b}:{i0:p,i1:S,b0:-b,b1:-g}}).sort((m,p)=>m.b0-p.b0),d=[];for(let m of l){let p=d.find(g=>{let u=g[g.length-1];return m.b0>u.b0&&m.b0-u.b1<Math.min(u.b1-u.b0,m.b1-m.b0)&&m.i0<u.i1&&m.i1>u.i0});p?p.push(m):d.push([m])}let c=(m,p)=>e.isVertical?`${p*h} ${m}`:`${m} ${p}`;return d.map(m=>{let p=m.length-1,g=S=>(m[S].b1+m[S+1].b0)/2,u=(S,b)=>g(S)+Math.sign(b)*a,f=[c(m[0].i1+a,m[0].b0-a)];for(let S=0;S<p;S++){let b=u(S,m[S].i1-m[S+1].i1);f.push(c(m[S].i1+a,b),c(m[S+1].i1+a,b))}f.push(c(m[p].i1+a,m[p].b1+a),c(m[p].i0-a,m[p].b1+a));for(let S=p-1;S>=0;S--){let b=u(S,m[S+1].i0-m[S].i0);f.push(c(m[S+1].i0-a,b),c(m[S].i0-a,b))}return f.push(c(m[0].i0-a,m[0].b0-a)),`M${f.join(" L")} Z`}).join(" ")}computeOverlayRects(t,e,i,o){if(t.decoration?.style?.layout==="bounds")return[o?{left:i.left-o,right:i.right+o,top:i.top-o,bottom:i.bottom+o,width:i.width+o*2,height:i.height+o*2}:i];let n=t.decoration.style.type,s=n===R.Underline||n===R.Strikethrough,a=n===R.Strikethrough,h=s?In(t.range,["rt","rp"]):t.range,l=Ct(h,!0,e.isVertical,s?0:o);return l=l.sort((d,c)=>e.isVertical?(e.isVertLR?1:-1)*(d.left-c.left):d.top-c.top),l.map(d=>{let c=d;if(a){let m=e.blockSize(d)*.1,p=e.blockStart(d)+e.blockSize(d)/2-m/2;c=e.isVertical?{left:p,right:p+m,top:d.top,bottom:d.bottom,width:m,height:d.height}:{top:p,bottom:p+m,left:d.left,right:d.right,height:m,width:d.width}}return o&&s&&(c=e.isVertical?{...c,top:c.top-o,bottom:c.bottom+o,height:c.height+o*2}:{...c,left:c.left-o,right:c.right+o,width:c.width+o*2}),c})}repositionItem(t,e){if(!t.container)return!0;let i=t.decoration.style;if(i.type!==R.Template){let d=i.type??R.Highlight;if(d===R.TextColor||d===R.Mask)return!0}let o=1/this.effectiveZoom(),n=i.expand??0,s=t.range.getBoundingClientRect(),a=(()=>{if(i.type!==R.Outline)return 0;let d=i.width;return d==="page"||d==="viewport"?Pe+ri:0})(),h=this.computeOverlayRects(t,e,s,n);if(r.isShapeOutline(i))return t.container.querySelector("path")?.setAttribute("d",this.outlinePath(t,e,o,h,s,a,er)),!0;let l=Array.from(t.container.children);return l.length!==h.length?!1:(l.forEach((d,c)=>this.positionElement(t,e,o,d,h[c],s,a)),!0)}layout(t){if(this.experimentalHighlights&&!this.notTextFlag?.has(t.id))return this.experimentalLayout(t);let e=this.wnd.document.createElement("div");e.setAttribute("id",t.id),e.dataset.highlightId=t.decoration.id,e.style.setProperty("pointer-events","none");let i=Ut(this.wnd),o=1/this.effectiveZoom(),n=t.decoration.style.expand??0,s=t.range.getBoundingClientRect(),a=t.decoration.style,h=(()=>{if(a.type!==R.Outline)return 0;let c=a.width;return c==="page"||c==="viewport"?Pe+ri:0})(),l;if(a.type===R.Template){a.stylesheet&&this.injectCustomStylesheet(a.stylesheet);let c=Qn(this.wnd,a.element);if(!c){t.container=e,t.clickableElements=[];return}c.style.setProperty("pointer-events","none"),l=c}else{let c=a,m=c.type??R.Highlight,p=c.tint??So(m);if(m===R.TextColor){t.container=e,t.clickableElements=[];return}if(m===R.Mask){t.container=e,t.clickableElements=[],this.scheduleMaskUpdate();return}let g=this.getCurrentDarkMode(),u=this.getBackgroundColor(),f=c.enforceContrast!==!1;if(r.isShapeOutline(c)){let C="http://www.w3.org/2000/svg",v=this.wnd.document.createElementNS(C,"svg");v.dataset.readium="true",v.style.cssText=`position:absolute;top:0;left:0;width:1px;height:1px;overflow:visible;pointer-events:none;z-index:${Ze}`;let w=this.wnd.document.createElementNS(C,"path");w.setAttribute("fill","none"),w.setAttribute("stroke",f?St(p,u):p),w.setAttribute("stroke-width",String(Pe)),w.setAttribute("d",this.outlinePath(t,i,o,this.computeOverlayRects(t,i,s,n),s,h,er)),v.append(w),e.append(v),t.container=e,t.clickableElements=[];return}let S=(()=>{switch(m){case R.Underline:{let v=f?St(p,u):p,w=c.layout==="bounds",[B,it]=i.isVertical?["border-right","border-left"]:["border-bottom","border-top"];return[w?`${it}: 0.1em solid ${v} !important`:null,`${B}: 0.1em solid ${v} !important`,"background-color: transparent !important","box-sizing: border-box !important"].filter(Boolean).join("; ")}case R.Strikethrough:{let v=f?St(p,u):p;return c.layout==="bounds"?[`background: repeating-linear-gradient(-45deg, transparent, transparent 19px, ${v} 19px, ${v} 20px) !important`,"background-color: transparent !important","box-sizing: border-box !important"].join("; "):[`background-color: ${v} !important`,"box-sizing: border-box !important"].join("; ")}case R.Outline:let C=f?St(p,u):p;return[`outline: ${Pe}px solid ${C} !important`,`outline-offset: ${ri}px !important`,"background-color: transparent !important","box-sizing: border-box !important"].join("; ");case R.HighlightUnderline:{let v=f?St(p,u):p,{r:w,g:B,b:it}=ft(v),lt=`rgba(${w}, ${B}, ${it}, 0.3)`,Di=c.layout==="bounds",[Ui,Ii]=i.isVertical?["border-right","border-left"]:["border-bottom","border-top"];return[`background-color: ${lt} !important`,Di?`${Ii}: 0.1em solid ${v} !important`:null,`${Ui}: 0.1em solid ${v} !important`,"box-sizing: border-box !important"].filter(Boolean).join("; ")}case R.Highlight:default:return[`background-color: ${f?St(p,u):p} !important`,`mix-blend-mode: ${g?"exclusion":"multiply"} !important`,"opacity: 1 !important","box-sizing: border-box !important","transform: translateZ(0) !important"].join("; ")}})(),b=this.wnd.document.createElement("template");b.innerHTML=`<div data-readium="true" class="readium-${m}" style="${S}"></div>`.trim(),l=b.content.firstElementChild}let d=this.computeOverlayRects(t,i,s,n);for(let c of d){let m=l.cloneNode(!0);m.style.setProperty("pointer-events","none"),this.positionElement(t,i,o,m,c,s,h),e.append(m)}t.container=e,t.clickableElements=Array.from(e.querySelectorAll("[data-activable='1']")),t.clickableElements.length||(t.clickableElements=Array.from(e.children))}renderLayout(t){t.forEach(e=>this.pendingRender.add(e)),this.wnd.cancelAnimationFrame(this.currentRender),this.currentRender=this.wnd.requestAnimationFrame(()=>{t=Array.from(this.pendingRender).filter(e=>this.items.includes(e)&&(!this.experimentalHighlights||!!this.notTextFlag?.has(e.id))),this.pendingRender.clear(),t.length!==0&&this.requireContainer().append(...t.map(e=>e.container).filter(e=>!!e))})}requireContainer(t=!1){if(t){let e;return this.wnd.document.getElementById(`${this.id}-style`)?e=this.wnd.document.getElementById(`${this.id}-style`):(e=this.wnd.document.createElement("style"),e.dataset.readium="true",e.id=`${this.id}-style`,this.wnd.document.head.appendChild(e)),e}return this.container||(this.shadowRoot||(this.shadowHost=this.wnd.document.createElement("div"),this.shadowHost.style.cssText="position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none",this.wnd.document.body.appendChild(this.shadowHost),this.shadowRoot=this.shadowHost.attachShadow({mode:"open"})),this.container=this.wnd.document.createElement("div"),this.container.setAttribute("id",this.id),this.container.dataset.group=this.name,this.container.dataset.readium="true",this.container.style.setProperty("pointer-events","none"),this.container.style.display="contents",this.shadowRoot.appendChild(this.container)),this.container}getCurrentDarkMode(){return po(this.wnd,"--USER__appearance")==="readium-night-on"||We(this.getBackgroundColor())}getBackgroundColor(){return po(this.wnd,"--USER__backgroundColor")||this.wnd.getComputedStyle(this.wnd.document.documentElement).getPropertyValue("background-color")}scheduleMaskUpdate(){this.maskFrame||(this.maskFrame=this.wnd.requestAnimationFrame(()=>{this.maskFrame=0,this.updateSharedMask()}))}removeShadowHost(){this.shadowHost?.remove(),this.shadowHost=void 0,this.shadowRoot=void 0}opaqueBackgroundColor(){let t=this.getBackgroundColor().trim();return!t||t==="transparent"||/^rgba\(.*,\s*(0|0?\.0+)\s*\)$|\/\s*(0|0?\.0+)\s*\)$/.test(t)?void 0:t}updateSharedMask(){let t=this.items.filter(p=>p.decoration.style?.type===R.Mask);if(t.length===0){this.maskSvg&&(this.maskSvg.remove(),this.maskSvg=void 0),this.container||this.removeShadowHost();return}this.maskSvg&&(this.maskSvg.style.display="none");let e=Ut(this.wnd),i=1/this.effectiveZoom(),o=this.wnd.document.documentElement,n=o.scrollWidth*i,s=o.scrollHeight*i,a=[];for(let p of t){let g=p.decoration.style,u=g.layout??"boxes",f=g.width??"wrap",S=g.expand??0,b=p.range.getBoundingClientRect(),C=u==="bounds"?[S?{left:b.left-S,top:b.top-S,right:b.right+S,bottom:b.bottom+S,width:b.width+S*2,height:b.height+S*2}:b]:Ct(p.range,!1,!1,S);for(let v of C){let w;switch(f){case"viewport":{let B=Math.floor(e.inlineStart(v)/e.viewportInlineSize)*e.viewportInlineSize;w=e.toRect(B,e.blockStart(v),e.viewportInlineSize,e.blockSize(v));break}case"page":{let B=Math.floor(e.inlineStart(v)/e.pageInlineSize)*e.pageInlineSize;w=e.toRect(B,e.blockStart(v),e.pageInlineSize,e.blockSize(v));break}case"bounds":{w=e.toRect(e.inlineStart(b),e.blockStart(v),e.inlineSize(b),e.blockSize(v));break}default:w=e.toRect(e.inlineStart(v),e.blockStart(v),e.inlineSize(v),e.blockSize(v))}a.push(w)}}let h=this.closeInterlines(e,a).map(p=>{let g=(p.left+e.xDocOffset)*i,u=(p.top+e.yDocOffset)*i,f=(p.right+e.xDocOffset)*i,S=(p.bottom+e.yDocOffset)*i;return`M${g} ${u} H${f} V${S} H${g} Z`}).join(" "),l="http://www.w3.org/2000/svg";if(!this.maskSvg){this.shadowRoot||(this.shadowHost=this.wnd.document.createElement("div"),this.shadowHost.style.cssText="position:absolute;top:0;left:0;width:100%;height:100%;pointer-events:none",this.wnd.document.body.appendChild(this.shadowHost),this.shadowRoot=this.shadowHost.attachShadow({mode:"open"})),this.maskSvg=this.wnd.document.createElementNS(l,"svg"),this.maskSvg.style.cssText=`position:absolute;top:0;left:0;width:${n}px;height:${s}px;pointer-events:none;z-index:${Ze-1}`,this.maskSvg.dataset.readium="true";let p=this.wnd.document.createElementNS(l,"defs"),g=this.wnd.document.createElementNS(l,"mask"),u=`${this.id}-mask-holes`;g.setAttribute("id",u),g.setAttribute("maskUnits","userSpaceOnUse");let f=this.wnd.document.createElementNS(l,"rect");f.setAttribute("fill","white");let S=this.wnd.document.createElementNS(l,"path");S.setAttribute("fill","black"),g.append(f,S),p.appendChild(g),this.maskSvg.appendChild(p);let b=this.wnd.document.createElementNS(l,"rect");b.setAttribute("id",`${this.id}-mask-rect`),b.setAttribute("mask",`url(#${u})`),b.style.pointerEvents="none",this.maskSvg.appendChild(b),this.shadowRoot.appendChild(this.maskSvg)}this.maskSvg.style.display="",this.maskSvg.style.width=`${n}px`,this.maskSvg.style.height=`${s}px`;let d=this.maskSvg.querySelector("mask > rect");d?.setAttribute("width",String(n)),d?.setAttribute("height",String(s));let c=this.maskSvg.querySelector("mask > path");c&&c.setAttribute("d",h);let m=this.maskSvg.querySelector(":scope > rect");if(m){let p=t[0].decoration.style.tint,g=p?void 0:this.opaqueBackgroundColor(),u=p??g??So(R.Mask),f=g?"0.5":"1";m.setAttribute("x","0"),m.setAttribute("y","0"),m.setAttribute("width",String(n)),m.setAttribute("height",String(s)),m.setAttribute("fill",u),m.setAttribute("fill-opacity",f)}}closeInterlines(t,e){let i=e.map(s=>({i0:t.inlineStart(s),i1:t.inlineStart(s)+t.inlineSize(s),b0:t.blockStart(s),b1:t.blockStart(s)+t.blockSize(s)})).sort((s,a)=>s.b0-a.b0),o=i.map(s=>s.b1),n=i.map(s=>s.b0);return i.forEach((s,a)=>{let h=i.flatMap((c,m)=>c.b0>=s.b1&&c.i0<s.i1&&c.i1>s.i0&&c.b0-s.b1<Math.min(s.b1-s.b0,c.b1-c.b0)?[m]:[]);if(h.length===0)return;let l=i[h[0]],d=(s.b1+l.b0)/2;o[a]=d;for(let c of h)i[c].b0<l.b1&&(n[c]=Math.min(n[c],d))}),i.map((s,a)=>t.toRect(s.i0,n[a],s.i1-s.i0,o[a]-n[a]))}injectCustomStylesheet(t){let e=`${this.id}-custom-style`,i=this.wnd.document.getElementById(e);i||(i=this.wnd.document.createElement("style"),i.id=e,i.dataset.readium="true",this.wnd.document.head.appendChild(i)),i.innerHTML=t}clearContainer(){if(this.experimentalHighlights){let t=this.wnd.CSS.highlights;for(let e of this._tintSubKeys.values())t.delete(e);this._tintSubKeys.clear(),this._subKeyCounter=0}this.wnd.document.getElementById(`${this.id}-custom-style`)?.remove(),this.container&&(this.container.remove(),this.container=void 0)}},nr=class de extends Q{constructor(){super(...arguments),this.observedResizeTargets=new Map,this.resizeFrame=0,this.lastGroupId=0,this.groups=new Map,this.handleResizer=this.handleResize.bind(this),this.scrollScheduled=!1,this.handleScroller=this.handleScroll.bind(this)}cleanup(){this.groups.forEach(t=>t.destroy()),this.groups.clear(),this.observedResizeTargets.clear()}updateHighlightStyles(){this.groups.forEach(t=>{t.requestLayout()})}handleResize(){this.wnd.clearTimeout(this.resizeFrame),this.resizeFrame=this.wnd.setTimeout(()=>{this.groups.forEach(t=>t.requestLayout())},50)}handleScroll(){this.scrollScheduled||(this.scrollScheduled=!0,this.wnd.requestAnimationFrame(()=>{this.scrollScheduled=!1,this.groups.forEach(t=>{t.hasOverlayItems&&t.repositionOverlays()})}))}mount(t,e){return this.wnd=t,e.register("decorate",de.moduleName,(i,o)=>{let n=i;(n.action==="add"||n.action==="update")&&n.decoration.locator&&(n.decoration.locator=z.deserialize(n.decoration.locator)),this.groups.has(n.group)||this.groups.set(n.group,new yo(t,e,`readium-decoration-${this.lastGroupId++}`,n.group));let s=this.groups.get(n.group);switch(n.action){case"add":s?.add(n.decoration);break;case"remove":s?.remove(n.decoration.id);break;case"clear":s?.clear();break;case"update":s?.update(n.decoration);break}o(!0)}),e.register("decoration_resize",de.moduleName,(i,o)=>{let n=i;n.action==="watch"?this.addDecorationResizeTarget(n.selector):this.removeDecorationResizeTarget(n.selector),o(!0)}),e.register("decoration_activatable",de.moduleName,(i,o)=>{let n=i,s=this.groups.get(n.group);s&&(s.activatable=n.activatable),o(!0)}),e.register("decoration_hoverable",de.moduleName,(i,o)=>{let n=i,s=this.groups.get(n.group);s&&(s.hoverable=n.hoverable),o(!0)}),this.resizeObserver=new ResizeObserver(()=>t.requestAnimationFrame(()=>this.handleResize())),this.resizeObserver.observe(t.document.documentElement),t.addEventListener("orientationchange",this.handleResizer),t.addEventListener("resize",this.handleResizer),t.addEventListener("scroll",this.handleScroller,{passive:!0,capture:!0}),this.styleObserver=new MutationObserver(i=>{i.some(o=>o.type==="attributes"&&o.attributeName==="style"&&o.oldValue!==o.target.getAttribute("style"))&&this.updateHighlightStyles()}),this.styleObserver.observe(t.document.documentElement,{attributes:!0,attributeFilter:["style"],attributeOldValue:!0}),e.log("Decorator Mounted"),!0}unmount(t,e){return t.removeEventListener("orientationchange",this.handleResizer),t.removeEventListener("resize",this.handleResizer),t.removeEventListener("scroll",this.handleScroller,{capture:!0}),e.unregisterAll(de.moduleName),this.resizeObserver.disconnect(),this.styleObserver.disconnect(),this.cleanup(),e.log("Decorator Unmounted"),!0}addDecorationResizeTarget(t){let e=this.observedResizeTargets.get(t);e&&this.resizeObserver.unobserve(e);let i=this.wnd.document.querySelector(t);i?(this.resizeObserver.observe(i),this.observedResizeTargets.set(t,i)):this.observedResizeTargets.delete(t)}removeDecorationResizeTarget(t){let e=this.observedResizeTargets.get(t);e&&(this.resizeObserver.unobserve(e),this.observedResizeTargets.delete(t))}};nr.moduleName="decorator";var _o=nr;var rr=class Bt extends Q{wndOnErr(t){this.comms?.send("error",{message:t.message,filename:t.filename,lineno:t.lineno,colno:t.colno})}mount(t,e){return this.comms=e,t.addEventListener("error",this.wndOnErr,!1),e.register("get_properties",Bt.moduleName,(i,o)=>{Ce(t),o(!0)}),e.register("update_properties",Bt.moduleName,(i,o)=>{ti(t,i),o(!0)}),e.register("set_property",Bt.moduleName,(i,o)=>{let n=i;Rt(t,n[0],n[1]),o(!0)}),e.register("remove_property",Bt.moduleName,(i,o)=>{It(t,i),o(!0)}),e.register("activate",Bt.moduleName,(i,o)=>{o(!0)}),e.log("WebPubSetup Mounted"),!0}unmount(t,e){return e.unregisterAll(Bt.moduleName),t.removeEventListener("error",this.wndOnErr),e.log("WebPubSetup Unmounted"),!0}};rr.moduleName="webpub_setup";var sr=rr;var ar=class bo extends Q{constructor(){super(...arguments),this.styleElement=null,this.beforePrintHandler=null,this.configApplied=!1}setupPrintProtection(t,e){if(!e.disable)return;let i=t.document.createElement("style");i.textContent=`
            @media print {
                body * {
                    display: none !important;
                }
                body::after {
                    content: "${e.watermark||"Printing has been disabled"}";
                    font-size: 200%;
                    display: block;
                    text-align: center;
                    margin-top: 50vh;
                    transform: translateY(-50%);
                }
            }
        `,t.document.head.appendChild(i),this.styleElement=i,this.beforePrintHandler=o=>(o.preventDefault(),!1),t.addEventListener("beforeprint",this.beforePrintHandler)}registerPrintHandlers(){this.comms?.register("print_protection",bo.moduleName,t=>{let e=t;return this.configApplied||(this.configApplied=!0,this.setupPrintProtection(this.wnd,e),this.comms?.log("Print protection configuration applied")),!0})}mount(t,e){return this.wnd=t,this.comms=e,this.registerPrintHandlers(),!0}unmount(t,e){return this.beforePrintHandler&&(t.removeEventListener("beforeprint",this.beforePrintHandler),this.beforePrintHandler=null),this.styleElement?.parentNode&&(this.styleElement.parentNode.removeChild(this.styleElement),this.styleElement=null),this.comms?.unregisterAll(bo.moduleName),this.configApplied=!1,!0}};ar.moduleName="print_protection";var lr=ar;var hr="readium-cjk-vertical-snapper-style",cr=class $ extends M{constructor(){super(...arguments),this.patternAnalyzer=null,this.lastScrollTime=0,this.isScrollProtectionEnabled=!1,this.initialScrollHandled=!1,this.isScrolling=!1,this.lastScrollLeft=0,this.isResizing=!1,this.resizeDebounce=null,this.verticalLR=!1,this.handleScroll=t=>{if(this.comms.ready&&!this.isResizing){if(!this.initialScrollHandled){this.lastScrollLeft=Math.abs(this.doc().scrollLeft),this.initialScrollHandled=!0,this.reportProgress();return}this.isScrolling||(this.isScrolling=!0,this.wnd.requestAnimationFrame(()=>{this.reportProgress();let e=Math.abs(this.doc().scrollLeft),i=e-this.lastScrollLeft;if(this.lastScrollLeft=e,this.isScrollProtectionEnabled&&Math.abs(i)>5){let o=Date.now(),n=o-(this.lastScrollTime||o);if(this.patternAnalyzer&&this.patternAnalyzer.analyze(i>0?"down":"up",Math.abs(i),n)){let s=t.target&&"tagName"in t.target?{tagName:t.target.tagName}:null;this.comms?.send("content_protection",{type:"suspicious_scrolling",timestamp:Date.now(),scrollDelta:i,scrollDirection:i>0?"left":"right",targetElement:s})}this.lastScrollTime=o}this.comms.send("scroll",i),this.isScrolling=!1}))}}}doc(){return this.wnd.document.scrollingElement}scrollable(){return Math.max(0,this.doc().scrollWidth-this.wnd.innerWidth)}hasScrolledPast(t){let e=t.getBoundingClientRect(),i=this.wnd.innerWidth/2,o=this.wnd.innerWidth*M.CENTER_TOLERANCE;return this.verticalLR?e.right<=i+o:e.left>=i-o}inCenterBand(t){let e=t.getBoundingClientRect(),i=this.wnd.innerWidth/2,o=this.wnd.innerWidth*M.CENTER_TOLERANCE,n=i-o,s=i+o;return e.width===0?e.left<=s&&e.left>=n:Math.max(0,Math.min(e.right,s)-Math.max(e.left,n))/e.width>=M.CENTER_TOLERANCE}fragmentStart(t){return t.getBoundingClientRect().left+Math.abs(this.doc().scrollLeft)}currentScrollExtent(){return{pos:Math.abs(this.doc().scrollLeft),size:this.wnd.innerWidth}}rectStart(t){let e=Math.abs(this.doc().scrollLeft);return this.verticalLR?t.left+e:e+this.wnd.innerWidth-t.right}reportProgress(t){if(!this.comms.ready)return;let e=this.doc().scrollWidth,i=this.wnd.innerWidth,o=Math.max(1,e-i),n=Math.abs(this.doc().scrollLeft),s=Math.max(0,Math.min(1,n/o)),a=Math.max(0,Math.min(1,(n+i)/e));this.comms.send("progress",{start:s,end:a,fragmentId:t!==void 0?t:this.currentTimelineFragment(),visibleFragmentIds:this.sortedVisibleFragmentIds()}),this.sendTextLayout()}enableScrollProtection(){this.patternAnalyzer||(this.patternAnalyzer=new pt(he),this.isScrollProtectionEnabled=!0,this.comms?.log("Scroll protection enabled"))}mount(t,e){this.wnd=t,this.comms=e,this.initialScrollHandled=!1,this.lastScrollLeft=0,this.isResizing=!1,this.verticalLR=co(t),this.resizeDebounce&&(this.wnd.clearTimeout(this.resizeDebounce),this.resizeDebounce=null),t.navigator.epubReadingSystem&&(t.navigator.epubReadingSystem.layoutStyle="scrolling");let i=t.document.createElement("style");return i.dataset.readium="true",i.id=hr,i.textContent=`
        * {
            scrollbar-width: none;
        }
        body::-webkit-scrollbar {
            display: none;
        }
        html {
            overflow-x: auto !important;
            overflow-y: hidden !important;
        }
        `,t.document.head.appendChild(i),this.resizeObserver=new ResizeObserver(()=>{this.resizeDebounce&&this.wnd.clearTimeout(this.resizeDebounce),this.isResizing=!0,this.resizeDebounce=this.wnd.setTimeout(()=>{this.isResizing=!1,this.resizeDebounce=null,this.refreshFragmentStarts(),this.reportProgress(),this.sendTextLayout(!0)},50)}),this.resizeObserver.observe(t.document.body),t.addEventListener("scroll",this.handleScroll,{passive:!0}),e.register("force_webkit_recalc",$.moduleName,()=>{le(this.wnd);let o=this.doc().scrollLeft;this.verticalLR?this.doc().scrollLeft=o>1?o-1:o+1:this.doc().scrollLeft=o<-1?o+1:o-1,this.doc().scrollLeft=o}),this.registerTextLayout(t,e,$.moduleName),this.registerNavigationLock(t,e,$.moduleName),e.register("go_progression",$.moduleName,(o,n)=>{let s=o;if(s<0||s>1){e.send("error",{message:"go_progression must be given a position from 0.0 to 1.0"}),n(!1);return}this.wnd.requestAnimationFrame(()=>{let a=this.scrollable()*s;this.doc().scrollLeft=this.verticalLR?a:-a,this.reportProgress(this.fragmentFromGeometry()),F(this.wnd),n(!0)})}),e.register("go_id",$.moduleName,(o,n)=>{let s=t.document.getElementById(o);if(!s){n(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollLeft+=s.getBoundingClientRect().left-t.innerWidth/2;let a=o;this.reportProgress(this.timelineEntries.has(a)?a:this.nearestPrecedingTimelineEntry(s)),F(this.wnd),n(!0)})}),e.register("go_text",$.moduleName,(o,n)=>{let s=M.rangeOf(t,o);if(!s){n(!1);return}this.wnd.requestAnimationFrame(()=>{this.doc().scrollLeft+=s.getBoundingClientRect().left-t.innerWidth/2,this.reportProgress(this.nearestPrecedingTimelineEntry(s.startContainer)),F(this.wnd),n(!0)})}),e.register("go_start",$.moduleName,(o,n)=>{if(this.doc().scrollLeft===0)return n(!1);this.doc().scrollLeft=0,this.reportProgress(this.firstFragmentIfReached()),n(!0)}),e.register("go_end",$.moduleName,(o,n)=>{if(Math.abs(this.doc().scrollLeft)===this.scrollable())return n(!1);this.doc().scrollLeft=this.verticalLR?this.scrollable():-this.scrollable(),this.reportProgress(this.sortedFragmentIds[this.sortedFragmentIds.length-1]),n(!0)}),e.register(["go_next","go_prev"],$.moduleName,(o,n)=>n(!1)),e.register("unfocus",$.moduleName,(o,n)=>{F(this.wnd),n(!0)}),e.register("scroll_protection",$.moduleName,(o,n)=>{this.enableScrollProtection(),n(!0)}),e.register("focus",$.moduleName,(o,n)=>{this.reportProgress(),n(!0)}),e.register("first_visible_locator",$.moduleName,(o,n)=>{let s=wt(t,!0);this.comms.send("first_visible_locator",s.serialize()),n(!0)}),e.register("timeline_entries",$.moduleName,(o,n)=>{this.updateTimelineEntries(Array.isArray(o)?o:[],t),n(!0)}),e.log("CJKVerticalSnapper Mounted"),!0}unmount(t,e){return e.unregisterAll($.moduleName),this.unwatchTextLayout(),this.unlockNavigation(t),this.resizeObserver.disconnect(),this.handleScroll&&t.removeEventListener("scroll",this.handleScroll),t.document.getElementById(hr)?.remove(),this.patternAnalyzer&&(this.patternAnalyzer.clear(),this.patternAnalyzer=null,this.isScrollProtectionEnabled=!1),this.timelineEntries.clear(),this.cachedFragmentIds=[],this.sortedFragmentIds=[],this.cachedFragmentStarts.clear(),e.log("CJKVerticalSnapper Unmounted"),!0}};cr.moduleName="cjk_vertical_snapper";var dr=cr;var vo=["fixed_setup","decorator","peripherals","print_protection"],Ro=["reflowable_setup","decorator","peripherals","column_snapper","scroll_snapper","cjk_vertical_snapper","print_protection"];var me=new Map([Jn,Kn,sr,Dn,Gn,_o,Ln,Nn,dr,lr].map(r=>[r.moduleName,r]));var bt=class{constructor(t=window,e=[]){this.loadedModules=[],this.wnd=t,this.comms=new Re(t);let i=[...new Set(e)];if(i.length){if(typeof t>"u")throw Error("Loader is not in a web browser");t.parent!==t&&this.comms.log("Loader is probably in a frame"),this.loadedModules=i.map(o=>{let n=this.loadModule(o);if(n)return n.mount(this.wnd,this.comms),n}).filter(o=>o!==void 0)}}loadModule(t){let e=me.get(t);return e===void 0?(this.comms.log(`Module "${name}" does not exist in the library`),e):new e}addModule(t){let e=this.loadModule(t);return!e||!e.mount(this.wnd,this.comms)?!1:(this.loadedModules.push(e),!0)}removeModule(t){let e=me.get(t);if(e===void 0)return this.comms.log(`Module "${t}" does not exist in the library`),!1;let i=this.loadedModules.findIndex(o=>o instanceof e);return i<0?!1:(this.loadedModules[i].unmount(this.wnd,this.comms),this.loadedModules.splice(i,1),!0)}destroy(){this.comms.destroy(),this.loadedModules.forEach(t=>t.unmount(this.wnd,this.comms)),this.loadedModules=[]}};var wo={type:"developer_tools",keyCombos:[{keyCode:73,meta:!0,alt:!0},{keyCode:73,ctrl:!0,shift:!0},{keyCode:74,meta:!0,alt:!0},{keyCode:74,ctrl:!0,shift:!0},{keyCode:85,meta:!0,alt:!0},{keyCode:67,meta:!0,alt:!0},{keyCode:67,meta:!0,shift:!0},{keyCode:67,ctrl:!0,shift:!0},{keyCode:65,meta:!0,alt:!0},{keyCode:84,meta:!0,shift:!0,alt:!0},{keyCode:67,shift:!0,alt:!0},{keyCode:123},{keyCode:123,shift:!0},{keyCode:123,ctrl:!0,shift:!0},{keyCode:123,meta:!0,alt:!0}]},Co={type:"select_all",keyCombos:[{keyCode:65,meta:!0},{keyCode:65,ctrl:!0}]},Eo={type:"print",keyCombos:[{keyCode:80,meta:!0},{keyCode:80,ctrl:!0},{keyCode:80,meta:!0,shift:!0},{keyCode:80,ctrl:!0,shift:!0},{keyCode:80,meta:!0,alt:!0},{keyCode:80,ctrl:!0,alt:!0}]},Po={type:"save",keyCombos:[{keyCode:83,meta:!0},{keyCode:83,ctrl:!0}]};var Fo=class{mergeKeyboardPeripherals(t,e=[]){let i=[],o=e.filter(s=>!["developer_tools","select_all","print","save"].includes(s.type));t.disableSelectAll&&i.push(Co),t.disableSave&&i.push(Po),t.monitorDevTools&&i.push(wo),t.protectPrinting?.disable&&i.push(Eo);let n=[...i];for(let s of o){let a=s.keyCombos.filter(h=>!i.some(l=>l.keyCombos.some(d=>h.keyCode===d.keyCode&&h.ctrl===d.ctrl&&h.shift===d.shift&&h.alt===d.alt&&h.meta===d.meta&&(n.includes(l)||h.condition===d.condition))));a.length>0&&i.push({...s,keyCombos:a})}return i}},si=class r extends Fo{constructor(){super(...arguments),this.viewportWatchers=new Set}watchViewport(t){return this.viewportWatchers.add(t),()=>{this.viewportWatchers.delete(t)}}notifyViewportChange(t){let e=this.viewport,i={readingOrder:[...e.readingOrder],progressions:new Map([...e.progressions].map(([o,n])=>[o,{...n}])),positions:e.positions?[...e.positions]:null};this.notifiedViewport&&r.sameViewport(this.notifiedViewport,i)||(this.notifiedViewport=i,t(i),this.viewportWatchers.forEach(o=>o(i)))}static sameViewport(t,e){let i=(o,n)=>o===n||!!o&&!!n&&o.length===n.length&&o.every((s,a)=>s===n[a]);return i(t.readingOrder,e.readingOrder)&&i(t.positions,e.positions)&&t.readingOrder.every(o=>t.progressions.get(o)?.start===e.progressions.get(o)?.start&&t.progressions.get(o)?.end===e.progressions.get(o)?.end)}goLeft(t=!1,e){this.readingProgression===E.ltr?this.goBackward(t,e):this.readingProgression===E.rtl&&this.goForward(t,e)}goRight(t=!1,e){this.readingProgression===E.ltr?this.goForward(t,e):this.readingProgression===E.rtl&&this.goBackward(t,e)}};var ur=new Set(Object.values(R));function xo(r,t){return r===R.TextColor?typeof window<"u"&&"Highlight"in window:ur.has(r)?!0:!!t?.[r]}function Fe(r,t){let{style:e}=r;if(e.type===R.Template){let i=e;return{...r,style:{...i,element:mr(i,r)}}}if(e.type&&t?.[e.type]){let i=t[e.type];return{...r,style:{type:R.Template,layout:i.layout,width:i.width,stylesheet:i.stylesheet,element:mr(i,r)}}}return r}function mr(r,t){return typeof r.element=="function"?r.element(t):r.element}function Sa(r,t){if(r.type!==t.type)return!1;if(r.type===R.Template){let o=r,n=t;return o.layout===n.layout&&o.width===n.width&&o.stylesheet===n.stylesheet}let e=r,i=t;return e.tint===i.tint&&e.layout===i.layout&&e.width===i.width&&(e.enforceContrast??!0)===(i.enforceContrast??!0)&&(e.expand??0)===(i.expand??0)}function ko(r,t){return r.locator.href===t.locator.href&&JSON.stringify(r.locator.locations?.serialize?.()??r.locator.locations)===JSON.stringify(t.locator.locations?.serialize?.()??t.locator.locations)&&JSON.stringify(r.locator.text??null)===JSON.stringify(t.locator.text??null)&&Sa(r.style,t.style)&&JSON.stringify(r.extras??null)===JSON.stringify(t.extras??null)}var ya=1e4,_a=5e3,jt=class{constructor(t,e){this.registry=new Map,this._ready=!1,this.listenerBuffer=[],this.visibleTime=0,this.visibleSince=document.hidden?void 0:performance.now(),this.visibilityHandler=this.handleVisibility.bind(this),this.handler=this.handle.bind(this),this.wnd=t,this.origin=e;try{this.channelId=window.crypto.randomUUID()}catch{this.channelId=ne()}this.startGc(),window.addEventListener("message",this.handler),this.send("_ping",void 0)}set listener(t){this.listenerBuffer.length>0&&this.listenerBuffer.forEach(e=>t(e[0],e[1])),this.listenerBuffer=[],this._listener=t}clearListener(){typeof this._listener=="function"&&(this._listener=void 0)}startGc(){this.gc===void 0&&(this.handleVisibility(),document.addEventListener("visibilitychange",this.visibilityHandler),this.gc=setInterval(()=>{let t=this.visibleNow();this.registry.forEach((e,i)=>{t-e.time>ya&&(console.warn(i,"event for",e.key,"was never handled!"),this.registry.delete(i))})},_a))}stopGc(){this.visibleTime=this.visibleNow(),this.visibleSince=void 0,document.removeEventListener("visibilitychange",this.visibilityHandler),clearInterval(this.gc),this.gc=void 0}visibleNow(){return this.visibleTime+(this.visibleSince===void 0?0:performance.now()-this.visibleSince)}handleVisibility(){document.hidden?(this.visibleTime=this.visibleNow(),this.visibleSince=void 0):this.visibleSince===void 0&&(this.visibleSince=performance.now())}halt(){this._ready=!1,window.removeEventListener("message",this.handler),this.stopGc(),this._listener=void 0,this.registry.clear()}resume(){window.addEventListener("message",this.handler),this.startGc(),this._ready=!0}handle(t){let e=t.data;if(!e._readium){console.warn("Ignoring",e);return}if(e._channel===this.channelId)switch(e.key){case"_ack":{if(!e.id)return;let i=this.registry.get(e.id);if(!i)return;this.registry.delete(e.id),i.cb(!!e.data);return}case"_pong":this._ready=!0;default:{if(!this.ready)return;typeof this._listener=="function"?this._listener(e.key,e.data):this.listenerBuffer.push([e.key,e.data])}}}get ready(){return this._ready}send(t,e,i,o=!1,n=[]){let s=ne();return i&&this.registry.set(s,{cb:i,time:this.visibleNow(),key:t}),this.wnd.postMessage({_readium:vt,_channel:this.channelId,id:s,data:e,key:t,strict:o},"/",n),s}};var Pt=class{constructor(t,e){this.config=t,this.onUpdate=e,this.unsubs=[],this.conditionValues=new Map}setup(){let t=!0,e=new Map;this.config.forEach(i=>i.keyCombos.forEach(o=>{if(o.condition){let n=e.get(o.condition)??[];n.push(o),e.set(o.condition,n)}})),e.forEach((i,o)=>{let n=o.subscribe(s=>{i.forEach(a=>this.conditionValues.set(a,s)),t||this.onUpdate(this.buildSerializable())});this.unsubs.push(n)}),t=!1,this.onUpdate(this.buildSerializable())}buildSerializable(){return this.config.map(t=>({...t,keyCombos:t.keyCombos.filter(e=>!e.condition||this.conditionValues.get(e)===!0).map(({condition:e,...i})=>i)})).filter(t=>t.keyCombos.length>0)}destroy(){this.unsubs.forEach(t=>t()),this.unsubs=[],this.conditionValues.clear()}};function ai(){let r=Promise.resolve();return t=>{let e=r.then(t);return r=e.catch(()=>{}),e}}var To,pr;function gr(){if(pr)return To;pr=1;function r(o){if(typeof o!="string")throw new TypeError("Path must be a string. Received "+JSON.stringify(o))}function t(o,n){for(var s="",a=0,h=-1,l=0,d,c=0;c<=o.length;++c){if(c<o.length)d=o.charCodeAt(c);else{if(d===47)break;d=47}if(d===47){if(!(h===c-1||l===1))if(h!==c-1&&l===2){if(s.length<2||a!==2||s.charCodeAt(s.length-1)!==46||s.charCodeAt(s.length-2)!==46){if(s.length>2){var m=s.lastIndexOf("/");if(m!==s.length-1){m===-1?(s="",a=0):(s=s.slice(0,m),a=s.length-1-s.lastIndexOf("/")),h=c,l=0;continue}}else if(s.length===2||s.length===1){s="",a=0,h=c,l=0;continue}}n&&(s.length>0?s+="/..":s="..",a=2)}else s.length>0?s+="/"+o.slice(h+1,c):s=o.slice(h+1,c),a=c-h-1;h=c,l=0}else d===46&&l!==-1?++l:l=-1}return s}function e(o,n){var s=n.dir||n.root,a=n.base||(n.name||"")+(n.ext||"");return s?s===n.root?s+a:s+o+a:a}var i={resolve:function(){for(var o="",n=!1,s,a=arguments.length-1;a>=-1&&!n;a--){var h;a>=0?h=arguments[a]:(s===void 0&&(s=process.cwd()),h=s),r(h),h.length!==0&&(o=h+"/"+o,n=h.charCodeAt(0)===47)}return o=t(o,!n),n?o.length>0?"/"+o:"/":o.length>0?o:"."},normalize:function(o){if(r(o),o.length===0)return".";var n=o.charCodeAt(0)===47,s=o.charCodeAt(o.length-1)===47;return o=t(o,!n),o.length===0&&!n&&(o="."),o.length>0&&s&&(o+="/"),n?"/"+o:o},isAbsolute:function(o){return r(o),o.length>0&&o.charCodeAt(0)===47},join:function(){if(arguments.length===0)return".";for(var o,n=0;n<arguments.length;++n){var s=arguments[n];r(s),s.length>0&&(o===void 0?o=s:o+="/"+s)}return o===void 0?".":i.normalize(o)},relative:function(o,n){if(r(o),r(n),o===n||(o=i.resolve(o),n=i.resolve(n),o===n))return"";for(var s=1;s<o.length&&o.charCodeAt(s)===47;++s);for(var a=o.length,h=a-s,l=1;l<n.length&&n.charCodeAt(l)===47;++l);for(var d=n.length,c=d-l,m=h<c?h:c,p=-1,g=0;g<=m;++g){if(g===m){if(c>m){if(n.charCodeAt(l+g)===47)return n.slice(l+g+1);if(g===0)return n.slice(l+g)}else h>m&&(o.charCodeAt(s+g)===47?p=g:g===0&&(p=0));break}var u=o.charCodeAt(s+g),f=n.charCodeAt(l+g);if(u!==f)break;u===47&&(p=g)}var S="";for(g=s+p+1;g<=a;++g)(g===a||o.charCodeAt(g)===47)&&(S.length===0?S+="..":S+="/..");return S.length>0?S+n.slice(l+p):(l+=p,n.charCodeAt(l)===47&&++l,n.slice(l))},_makeLong:function(o){return o},dirname:function(o){if(r(o),o.length===0)return".";for(var n=o.charCodeAt(0),s=n===47,a=-1,h=!0,l=o.length-1;l>=1;--l)if(n=o.charCodeAt(l),n===47){if(!h){a=l;break}}else h=!1;return a===-1?s?"/":".":s&&a===1?"//":o.slice(0,a)},basename:function(o,n){if(n!==void 0&&typeof n!="string")throw new TypeError('"ext" argument must be a string');r(o);var s=0,a=-1,h=!0,l;if(n!==void 0&&n.length>0&&n.length<=o.length){if(n.length===o.length&&n===o)return"";var d=n.length-1,c=-1;for(l=o.length-1;l>=0;--l){var m=o.charCodeAt(l);if(m===47){if(!h){s=l+1;break}}else c===-1&&(h=!1,c=l+1),d>=0&&(m===n.charCodeAt(d)?--d===-1&&(a=l):(d=-1,a=c))}return s===a?a=c:a===-1&&(a=o.length),o.slice(s,a)}else{for(l=o.length-1;l>=0;--l)if(o.charCodeAt(l)===47){if(!h){s=l+1;break}}else a===-1&&(h=!1,a=l+1);return a===-1?"":o.slice(s,a)}},extname:function(o){r(o);for(var n=-1,s=0,a=-1,h=!0,l=0,d=o.length-1;d>=0;--d){var c=o.charCodeAt(d);if(c===47){if(!h){s=d+1;break}continue}a===-1&&(h=!1,a=d+1),c===46?n===-1?n=d:l!==1&&(l=1):n!==-1&&(l=-1)}return n===-1||a===-1||l===0||l===1&&n===a-1&&n===s+1?"":o.slice(n,a)},format:function(o){if(o===null||typeof o!="object")throw new TypeError('The "pathObject" argument must be of type Object. Received type '+typeof o);return e("/",o)},parse:function(o){r(o);var n={root:"",dir:"",base:"",ext:"",name:""};if(o.length===0)return n;var s=o.charCodeAt(0),a=s===47,h;a?(n.root="/",h=1):h=0;for(var l=-1,d=0,c=-1,m=!0,p=o.length-1,g=0;p>=h;--p){if(s=o.charCodeAt(p),s===47){if(!m){d=p+1;break}continue}c===-1&&(m=!1,c=p+1),s===46?l===-1?l=p:g!==1&&(g=1):l!==-1&&(g=-1)}return l===-1||c===-1||g===0||g===1&&l===c-1&&l===d+1?c!==-1&&(d===0&&a?n.base=n.name=o.slice(1,c):n.base=n.name=o.slice(d,c)):(d===0&&a?(n.name=o.slice(1,l),n.base=o.slice(1,c)):(n.name=o.slice(d,l),n.base=o.slice(d,c)),n.ext=o.slice(l,c)),d>0?n.dir=o.slice(0,d-1):a&&(n.dir="/"),n},sep:"/",delimiter:":",win32:null,posix:null};return i.posix=i,To=i,To}var zo=gr();function Ft(r){let t=r.languages?.[0]?.toLowerCase(),e=r.readingProgression;if(t){if(t.startsWith("zh")||t.startsWith("ja")||t.startsWith("ko"))return e===E.rtl?"cjk-vertical":"cjk-horizontal";if(t.startsWith("mn-mong"))return"mongolian-vertical";if(t.startsWith("ar")||t.startsWith("fa")||t.startsWith("he"))return"rtl"}return"ltr"}var ba={description:"Attempts to filter out paragraphs that are implicitly headings or part of headers",scope:"RS",value:"readium-experimentalHeaderFiltering-on"},va={description:"Attempts to filter out elements that are sized using viewport units and should not be scaled directly e.g. tables, images, iframes, etc.",scope:"RS",value:"readium-experimentalZoom-on"},fr={experimentalHeaderFiltering:ba,experimentalZoom:va};var li=fr;var xt=(r=>(r.start="start",r.left="left",r.right="right",r.justify="justify",r))(xt||{}),Gt={range:[0,100],step:1},Vt={range:[.7,4],step:.05},Xt={range:[100,1e3],step:100},$t={range:[50,250],step:10},Lo={range:[0,1],step:.125},Oo={range:[1,2.5],step:.1},Kt={range:[20,100],step:1},Mo={range:[0,3],step:.25},No={range:[0,3],step:.25},Ao={range:[0,2],step:.125};var xe=class{constructor(){}toFlag(t){return`readium-${t}-on`}toUnitless(t){return t.toString()}toPercentage(t,e=!1){return e||t>0&&t<=1?`${Math.round(t*100)}%`:`${t}%`}toVw(t){let e=Math.round(t*100);return`${Math.min(e,100)}vw`}toVh(t){let e=Math.round(t*100);return`${Math.min(e,100)}vh`}toPx(t){return`${t}px`}toRem(t){return`${t}rem`}};function Sr(r,t){return r==null||t==null||r<=t?r:void 0}function yr(r,t){return r==null||t==null||r>=t?r:void 0}function K(r){return typeof r=="string"?r:r===null?null:void 0}function L(r){return typeof r=="boolean"||r==null?r:void 0}function hi(r,t){if(r!==void 0)return r===null?null:t[r]!==void 0?r:void 0}function kt(r){return typeof r=="boolean"||typeof r=="number"&&r>=0?r:r===null?null:void 0}function P(r){if(r!==void 0)return r===null?null:r<0?void 0:r}function Tt(r,t){if(r===void 0)return;if(r===null)return null;let e=Math.min(...t),i=Math.max(...t);return r>=e&&r<=i?r:void 0}function ci(r,t){return r===void 0?t:r}function _r(r){if(r!==void 0)return r===null?null:r.filter(t=>t in li)}var N=class{constructor({initialValue:t=null,effectiveValue:e,isEffective:i,onChange:o}){this._value=t,this._effectiveValue=e,this._isEffective=i,this._onChange=o}set value(t){this._value=t,this._onChange(this._value)}get value(){return this._value}get effectiveValue(){return this._effectiveValue}get isEffective(){return this._isEffective}clear(){this._value=null}},tt=class extends N{set value(t){this._value=t,this._onChange(this._value)}get value(){return this._value}get effectiveValue(){return this._effectiveValue}get isEffective(){return this._isEffective}clear(){this._value=null}toggle(){this._value=!this._value,this._onChange(this._value)}},di=class extends N{constructor({initialValue:t=null,effectiveValue:e,isEffective:i,onChange:o,supportedValues:n}){super({initialValue:t,effectiveValue:e,isEffective:i,onChange:o}),this._supportedValues=n}set value(t){if(t&&!this._supportedValues.includes(t))throw new Error(`Value '${String(t)}' is not in the supported values for this preference.`);this._value=t,this._onChange(this._value)}get value(){return this._value}get effectiveValue(){return this._effectiveValue}get isEffective(){return this._isEffective}get supportedValues(){return this._supportedValues}clear(){this._value=null}},W=class extends N{constructor({initialValue:t=null,effectiveValue:e,isEffective:i,onChange:o,supportedRange:n,step:s}){super({initialValue:t,effectiveValue:e,isEffective:i,onChange:o}),this._supportedRange=n,this._step=s,this._decimals=this._step.toString().includes(".")?this._step.toString().split(".")[1].length:0}set value(t){if(t&&(t<this._supportedRange[0]||t>this._supportedRange[1]))throw new Error(`Value '${String(t)}' is out of the supported range for this preference.`);this._value=t,this._onChange(this._value)}get value(){return this._value}get effectiveValue(){return this._effectiveValue}get isEffective(){return this._isEffective}get supportedRange(){return this._supportedRange}get step(){return this._step}increment(){this._value&&this._value<this._supportedRange[1]&&(this._value=Math.min(Math.round((this._value+this._step)*10**this._decimals)/10**this._decimals,this._supportedRange[1]),this._onChange(this._value))}decrement(){this._value&&this._value>this._supportedRange[0]&&(this._value=Math.max(Math.round((this._value-this._step)*10**this._decimals)/10**this._decimals,this._supportedRange[0]),this._onChange(this._value))}format(t){return t.toString()}clear(){this._value=null}};var vr=r=>{if("blob"in r&&r.blob.type)return r.blob.type;if(r.as==="script")return"text/javascript";if(r.as==="link"&&"url"in r){let t=r.url.toLowerCase();if(t.endsWith(".css"))return"text/css";if([".js",".mjs",".cjs"].some(e=>t.endsWith(e)))return"text/javascript"}},Rr=(r,t)=>{t.attributes&&Object.entries(t.attributes).forEach(([e,i])=>{e==="type"||e==="rel"||e==="href"||e==="src"||i!=null&&(typeof i=="boolean"?i&&r.setAttribute(e,""):r.setAttribute(e,i))})},Ra=(r,t,e)=>{let i=r.createElement("script");i.dataset.readium="true",t.id&&(i.id=t.id);let o=t.type||vr(t);return o&&(i.type=o),Rr(i,t),i.src=e,i},br=(r,t,e)=>{let i=r.createElement("link");i.dataset.readium="true",t.id&&(i.id=t.id),t.rel&&(i.rel=t.rel);let o=t.type||vr(t);return o&&(i.type=o),Rr(i,t),i.href=e,i},mi=class{constructor(t){this.blobStore=new Map,this.createdBlobUrls=new Set,this.allowedDomains=[],this.injectableIdCounter=0,this.allowedDomains=(t.allowedDomains||[]).map(e=>{try{return new URL(e),e}catch{throw new Error(`Invalid allowed domain: "${e}". Must be a valid URL (e.g., "https://fonts.googleapis.com").`)}}),this.rules=t.rules.map(e=>{let i={...e};return e.prepend&&(i.prepend=e.prepend.map(o=>({...o,id:o.id||`injectable-${this.injectableIdCounter++}`})).reverse()),e.append&&(i.append=e.append.map(o=>({...o,id:o.id||`injectable-${this.injectableIdCounter++}`}))),i})}dispose(){for(let t of this.createdBlobUrls)try{URL.revokeObjectURL(t)}catch(e){console.warn("Failed to revoke blob URL:",t,e)}this.createdBlobUrls.clear()}getAllowedDomains(){return[...this.allowedDomains]}async injectForDocument(t,e){for(let i of this.rules)this.matchesRule(i,e)&&await this.applyRule(t,i)}matchesRule(t,e){let i=e.href;return t.resources.some(o=>o instanceof RegExp?o.test(i):i===o)}async getOrCreateBlobUrl(t){let e=t.id;if(this.blobStore.has(e)){let i=this.blobStore.get(e);return i.refCount++,i.url}if("blob"in t){let i=URL.createObjectURL(t.blob);return this.blobStore.set(e,{url:i,refCount:1}),this.createdBlobUrls.add(i),i}throw new Error("Resource must have a blob property")}async releaseBlobUrl(t){if(!this.createdBlobUrls.has(t))return;let e=Array.from(this.blobStore.values()).find(i=>i.url===t);if(e&&(e.refCount--,e.refCount<=0)){URL.revokeObjectURL(t),this.createdBlobUrls.delete(t);for(let[i,o]of this.blobStore.entries())if(o.url===t){this.blobStore.delete(i);break}}}async getResourceUrl(t,e){if("url"in t){let i=new URL(t.url,e.baseURI).toString();if(!this.isValidUrl(i,e))throw new Error(`Invalid URL: Only HTTPS, data:, blob:, or localhost HTTP URLs are allowed. Got: ${t.url}`);return i}else return this.getOrCreateBlobUrl(t)}createPreloadLink(t,e,i){if(e.as!=="link"||e.rel!=="preload")return;let o={...e,rel:"preload",attributes:{...e.attributes,as:e.as}},n=br(t,o,i);t.head.appendChild(n)}createElement(t,e,i){if(e.as==="script")return Ra(t,e,i);if(e.as==="link")return br(t,e,i);throw new Error(`Unsupported element type: ${e.as}`)}async applyRule(t,e){let i=[],o=e.prepend?e.prepend.filter(s=>!s.condition||s.condition(t)):[],n=e.append?e.append.filter(s=>!s.condition||s.condition(t)):[];try{for(let s of o)await this.processInjectable(s,t,i,"prepend");for(let s of n)await this.processInjectable(s,t,i,"append")}catch(s){for(let{element:a,url:h}of i)try{a.remove(),await this.releaseBlobUrl(h)}catch(l){console.error("Error during cleanup:",l)}throw s}}async processInjectable(t,e,i,o){let n=t.target==="body"?e.body:e.head;if(!n)return;let s=null;try{if(s=await this.getResourceUrl(t,e),t.rel==="preload"&&"url"in t)this.createPreloadLink(e,t,s);else{let a=this.createElement(e,t,s);i.push({element:a,url:s}),o==="prepend"?n.prepend(a):n.append(a)}}catch(a){throw console.error("Failed to process resource:",a),s&&"blob"in t&&await this.releaseBlobUrl(s),a}}isValidUrl(t,e){try{let i=new URL(t,e.baseURI);if(i.protocol==="data:"||i.protocol==="blob:"&&this.createdBlobUrls.has(t))return!0;if(this.allowedDomains.length>0){let o=i.origin;return this.allowedDomains.some(n=>{let s=new URL(n).origin;return o===s})}return!1}catch{return!1}}};var ui=r=>r.replace(/\/\/.*/g,"").replace(/\/\*[\s\S]*?\*\//g,"").replace(/\n/g,"").replace(/\s+/g," "),ke=r=>r.replace(/\/\*(?:(?!\*\/)[\s\S])*\*\/|[\r\n\t]+/g,"").replace(/ {2,}/g," ");var wr='!function(t,e){"object"==typeof exports&&"object"==typeof module?module.exports=e():"function"==typeof define&&define.amd?define([],e):"object"==typeof exports?exports._readium_cssSelectorGenerator=e():t._readium_cssSelectorGenerator=e()}(self,(()=>(()=>{"use strict";var t={d:(e,n)=>{for(var o in n)t.o(n,o)&&!t.o(e,o)&&Object.defineProperty(e,o,{enumerable:!0,get:n[o]})},o:(t,e)=>Object.prototype.hasOwnProperty.call(t,e),r:t=>{"undefined"!=typeof Symbol&&Symbol.toStringTag&&Object.defineProperty(t,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(t,"__esModule",{value:!0})}},e={};function n(t){return"object"==typeof t&&null!==t&&t.nodeType===Node.ELEMENT_NODE}t.r(e),t.d(e,{_readium_cssSelectorGenerator:()=>Z,default:()=>tt,getCssSelector:()=>X});const o={NONE:"",DESCENDANT:" ",CHILD:" > "},r={id:"id",class:"class",tag:"tag",attribute:"attribute",nthchild:"nthchild",nthoftype:"nthoftype"},i="_readium_cssSelectorGenerator";function c(t="unknown problem",...e){console.warn(`${i}: ${t}`,...e)}const s={selectors:[r.id,r.class,r.tag,r.attribute],includeTag:!1,whitelist:[],blacklist:[],combineWithinSelector:!0,combineBetweenSelectors:!0,root:null,maxCombinations:Number.POSITIVE_INFINITY,maxCandidates:Number.POSITIVE_INFINITY,useScope:!1};function u(t){return t instanceof RegExp}function l(t){return["string","function"].includes(typeof t)||u(t)}function a(t){return Array.isArray(t)?t.filter(l):[]}function f(t){const e=[Node.DOCUMENT_NODE,Node.DOCUMENT_FRAGMENT_NODE,Node.ELEMENT_NODE];return function(t){return t instanceof Node}(t)&&e.includes(t.nodeType)}function d(t,e){if(f(t))return t.contains(e)||c("element root mismatch","Provided root does not contain the element. This will most likely result in producing a fallback selector using element\'s real root node. If you plan to use the selector using provided root (e.g. `root.querySelector`), it will not work as intended."),t;const n=e.getRootNode({composed:!1});return f(n)?(n!==document&&c("shadow root inferred","You did not provide a root and the element is a child of Shadow DOM. This will produce a selector using ShadowRoot as a root. If you plan to use the selector using document as a root (e.g. `document.querySelector`), it will not work as intended."),n):S(e)}function m(t){return"number"==typeof t?t:Number.POSITIVE_INFINITY}function p(t=[]){const[e=[],...n]=t;return 0===n.length?e:n.reduce(((t,e)=>t.filter((t=>e.includes(t)))),e)}function g(t){const e=t.map((t=>{if(u(t))return e=>t.test(e);if("function"==typeof t)return e=>{const n=t(e);return"boolean"!=typeof n?(c("pattern matcher function invalid","Provided pattern matching function does not return boolean. It\'s result will be ignored.",t),!1):n};if("string"==typeof t){const e=new RegExp("^"+t.replace(/[|\\\\{}()[\\]^$+?.]/g,"\\\\$&").replace(/\\*/g,".+")+"$");return t=>e.test(t)}return c("pattern matcher invalid","Pattern matching only accepts strings, regular expressions and/or functions. This item is invalid and will be ignored.",t),()=>!1}));return t=>e.some((e=>e(t)))}function h(t,e,n){const o=Array.from(d(n,t[0]).querySelectorAll(e));return o.length===t.length&&t.every((t=>o.includes(t)))}function y(t,e){e=null!=e?e:S(t);const o=[];let r=t;for(;n(r)&&r!==e;)o.push(r),r=r.parentElement;return o}function b(t,e){return p(t.map((t=>y(t,e))))}function S(t){return t.ownerDocument.querySelector(":root")}const N=", ",v=new RegExp(["^$","\\\\s"].join("|")),E=new RegExp(["^$"].join("|")),x=[r.nthoftype,r.tag,r.id,r.class,r.attribute,r.nthchild],w=g(["class","id","ng-*"]);function I({name:t}){return`[${t}]`}function T({name:t,value:e}){return`[${t}=\'${e}\']`}function O({nodeName:t,nodeValue:e}){return{name:F(t),value:F(null!=e?e:void 0)}}function C(t){const e=Array.from(t.attributes).filter((e=>function({nodeName:t,nodeValue:e},n){const o=n.tagName.toLowerCase();return!(["input","option"].includes(o)&&"value"===t||"src"===t&&(null==e?void 0:e.startsWith("data:"))||w(t))}(e,t))).map(O);return[...e.map(I),...e.map(T)]}function j(t){var e;return(null!==(e=t.getAttribute("class"))&&void 0!==e?e:"").trim().split(/\\s+/).filter((t=>!E.test(t))).map((t=>`.${F(t)}`))}function A(t){var e;const n=null!==(e=t.getAttribute("id"))&&void 0!==e?e:"",o=`#${F(n)}`,r=t.getRootNode({composed:!1});return!v.test(n)&&h([t],o,r)?[o]:[]}function R(t){var e;const n=null===(e=t.parentElement)||void 0===e?void 0:e.children;if(n)for(let e=0;e<n.length;e++)if(n[e]===t)return[`:nth-child(${String(e+1)})`];return[]}function $(t){return[F(t.tagName.toLowerCase())]}function D(t){const e=[...new Set((n=t.map($),[].concat(...n)))];var n;return 0===e.length||e.length>1?[]:[e[0]]}function k(t){const e=D([t])[0],n=t.parentElement;if(n){const o=Array.from(n.children).filter((t=>t.tagName.toLowerCase()===e)),r=o.indexOf(t);if(r>-1)return[`${e}:nth-of-type(${String(r+1)})`]}return[]}function*P(t=[],{maxResults:e=Number.POSITIVE_INFINITY}={}){let n=0,o=L(1);for(;o.length<=t.length&&n<e;){n+=1;const e=o.map((e=>t[e]));yield e,o=_(o,t.length-1)}}function _(t=[],e=0){const n=t.length;if(0===n)return[];const o=[...t];o[n-1]+=1;for(let t=n-1;t>=0;t--)if(o[t]>e){if(0===t)return L(n+1);o[t-1]++,o[t]=o[t-1]+1}return o[n-1]>e?L(n+1):o}function L(t=1){return Array.from(Array(t).keys())}const M=":".charCodeAt(0).toString(16).toUpperCase(),V=/[ !"#$%&\'()\\[\\]{|}<>*+,./;=?@^`~\\\\]/;function F(t=""){return CSS?CSS.escape(t):function(t=""){return t.split("").map((t=>":"===t?`\\\\${M} `:V.test(t)?`\\\\${t}`:escape(t).replace(/%/g,"\\\\"))).join("")}(t)}const Y={tag:D,id:function(t){return 0===t.length||t.length>1?[]:A(t[0])},class:function(t){return p(t.map(j))},attribute:function(t){return p(t.map(C))},nthchild:function(t){return p(t.map(R))},nthoftype:function(t){return p(t.map(k))}},G={tag:$,id:A,class:j,attribute:C,nthchild:R,nthoftype:k};function W(t){return t.includes(r.tag)||t.includes(r.nthoftype)?[...t]:[...t,r.tag]}function*q(t,e){const n={};for(const o of t){const t=e[o];t&&t.length>0&&(n[o]=t)}for(const t of function*(t={}){const e=Object.entries(t);if(0===e.length)return;const n=[{index:e.length-1,partial:{}}];for(;n.length>0;){const t=n.pop();if(!t)break;const{index:o,partial:r}=t;if(o<0){yield r;continue}const[i,c]=e[o];for(let t=c.length-1;t>=0;t--)n.push({index:o-1,partial:Object.assign(Object.assign({},r),{[i]:c[t]})})}}(n))yield B(t)}function B(t={}){const e=[...x];return t[r.tag]&&t[r.nthoftype]&&e.splice(e.indexOf(r.tag),1),e.map((e=>{return(o=t)[n=e]?o[n].join(""):"";var n,o})).join("")}function H(t,e){return[...t.map((t=>e+o.DESCENDANT+t)),...t.map((t=>e+o.CHILD+t))]}function*U(t,e,n="",o){const r=function*(t,e){const n=new Set,o=function(t,e){const{blacklist:n,whitelist:o,combineWithinSelector:r,maxCombinations:i}=e,c=g(n),s=g(o);return function(t){const{selectors:e,includeTag:n}=t,o=[...e];return n&&!o.includes("tag")&&o.push("tag"),o}(e).reduce(((e,n)=>{const o=function(t,e){return(0,Y[e])(t)}(t,n),u=function(t=[],e,n){return t.filter((t=>n(t)||!e(t)))}(o,c,s),l=function(t=[],e){return t.sort(((t,n)=>{const o=e(t),r=e(n);return o&&!r?-1:!o&&r?1:0}))}(u,s);return e[n]=r?Array.from(P(l,{maxResults:i})):l.map((t=>[t])),e}),{})}(t,e);for(const t of function*(t,e){for(const n of function(t){const{selectors:e,combineBetweenSelectors:n,includeTag:o,maxCandidates:r}=t,i=n?function(t=[],{maxResults:e=Number.POSITIVE_INFINITY}={}){return Array.from(P(t,{maxResults:e}))}(e,{maxResults:r}):e.map((t=>[t]));return o?i.map(W):i}(e))yield*q(n,t)}(o,e))n.has(t)||(n.add(t),yield t)}(t,o);for(const o of function*(t,e){if(""===e)yield*t;else for(const n of t)yield*H([n],e)}(r,n))h(t,o,e)&&(yield o)}function*z(t,e,n="",o){if(0===t.length)return null;const r=[t.length>1?t:[],...b(t,e).map((t=>[t]))];for(const t of r)for(const r of U(t,e,n,o))yield{foundElements:t,selector:r}}function J(t){return{value:t,include:!1}}function K({selectors:t,operator:e}){let n=[...x];t[r.tag]&&t[r.nthoftype]&&(n=n.filter((t=>t!==r.tag)));let o="";return n.forEach((e=>{var n;(null!==(n=t[e])&&void 0!==n?n:[]).forEach((({value:t,include:e})=>{e&&(o+=t)}))})),e+o}function Q(t,e){return t.map((t=>function(t,e){return[e?":scope":":root",...y(t,e).reverse().map((t=>{var e;const n=function(t,e,n=o.NONE){const r={};return e.forEach((e=>{Reflect.set(r,e,function(t,e){return G[e](t)}(t,e).map(J))})),{element:t,operator:n,selectors:r}}(t,[r.nthchild],o.CHILD);return(null!==(e=n.selectors.nthchild)&&void 0!==e?e:[]).forEach((t=>{t.include=!0})),n})).map(K)].join("")}(t,e))).join(N)}function X(t,e={}){return Z(t,Object.assign(Object.assign({},e),{maxResults:1})).next().value}function*Z(t,e={}){var o;const i=function(t){(t instanceof NodeList||t instanceof HTMLCollection)&&(t=Array.from(t));const e=(Array.isArray(t)?t:[t]).filter(n);return[...new Set(e)]}(t),c=function(t,e={}){const n=Object.assign(Object.assign({},s),e);return{selectors:(o=n.selectors,Array.isArray(o)?o.filter((t=>{return e=r,n=t,Object.values(e).includes(n);var e,n})):[]),whitelist:a(n.whitelist),blacklist:a(n.blacklist),root:d(n.root,t),combineWithinSelector:!!n.combineWithinSelector,combineBetweenSelectors:!!n.combineBetweenSelectors,includeTag:!!n.includeTag,maxCombinations:m(n.maxCombinations),maxCandidates:m(n.maxCandidates),useScope:!!n.useScope,maxResults:m(n.maxResults)};var o}(i[0],e),u=null!==(o=c.root)&&void 0!==o?o:S(i[0]);let l=0;for(const t of function*({elements:t,root:e,rootSelector:n="",options:o}){let r=e,i=n,c=!0;for(;c;){let n=!1;for(const c of z(t,r,i,o)){const{foundElements:o,selector:s}=c;if(n=!0,!h(t,s,e)){r=o[0],i=s;break}yield s}n||(c=!1)}}({elements:i,options:c,root:u,rootSelector:""}))if(yield t,l++,l>=c.maxResults)return;i.length>1&&(yield i.map((t=>X(t,c))).join(N),l++,l>=c.maxResults)||(yield Q(i,c.useScope?u:void 0))}const tt=X;return e})()));';var Cr=`(function() {
    if(window.onload) window.onload = new Proxy(window.onload, {
        apply: function(target, receiver, args) {
            if(!window._readium_blockEvents) {
                Reflect.apply(target, receiver, args);
                return;
            }
            _readium_blockedEvents.push([
                0, target, receiver, args
            ]);
        }
    });
})();
`;var pi=class{constructor(t){if(this.detectedTools=new Set,!t.onDetected)throw new Error("onDetected callback is required");this.options=t,this.setupDetection()}isAutomationToolPresent(){let t=window;return t.domAutomation||t.domAutomationController?"Selenium":navigator.webdriver===!0?"Puppeteer/Playwright":t.__webdriver_evaluate||t.__selenium_evaluate?"Chrome Automation":t.callPhantom||t._phantom?"PhantomJS":t.__nightmare?"Nightmare":t.$testCafe?"TestCafe":null}setupDetection(){let t=this.isAutomationToolPresent();if(t){this.handleDetected(t);return}this.observer=new MutationObserver(()=>{let e=this.isAutomationToolPresent();e&&!this.detectedTools.has(e)&&this.handleDetected(e)}),this.observer.observe(document.documentElement,{childList:!0,subtree:!0,attributes:!0}),window.addEventListener("unload",()=>this.destroy())}handleDetected(t){this.detectedTools.add(t),this.options.onDetected?.(t)}destroy(){this.observer?.disconnect(),this.observer=void 0,this.detectedTools.clear()}};var wa=0;function Ca(){return++wa}var Ea=`
onmessage = function(event) {
  var action = event.data;
  var startTime = performance.now()

  console[action.type](...action.payload);
  postMessage({
    id: action.id,
    time: performance.now() - startTime
  })
}
`,Er=class{constructor(t,e){this.callbacks=new Map,this.worker=t,this.blobUrl=e,this.worker.onmessage=i=>{let o=i.data,n=o.id,s=this.callbacks.get(o.id);s&&(s({time:o.time}),this.callbacks.delete(n))},this.log=(...i)=>this.send("log",...i),this.table=(...i)=>this.send("table",...i),this.clear=(...i)=>this.send("clear",...i)}async send(t,...e){let i=Ca();return new Promise((o,n)=>{this.callbacks.set(i,o),this.worker.postMessage({id:i,type:t,payload:e}),setTimeout(()=>{n(new Error("timeout")),this.callbacks.delete(i)},2e3)})}destroy(){this.worker.terminate(),URL.revokeObjectURL(this.blobUrl)}};Er.workerScript=Ea;var Do=Er;function Uo(r){return typeof window<"u"&&console?console[r]:(...t)=>{}}var Pr=Uo("log"),Io=Uo("table"),Fr=Uo("clear");async function Ho(){if(typeof navigator<"u"&&navigator.brave&&navigator.brave.isBrave)try{return await Promise.race([navigator.brave.isBrave(),new Promise(r=>setTimeout(()=>r(!1),1e3))])}catch{return!0}return!1}function xr(r){return r.excludes.some(t=>t())?!1:r.includes.some(t=>t())}var gi=class{constructor(t={}){if(this.isOpen=!1,this.checkCount=0,this.maxChecks=10,this.maxPrintTime=0,this.largeObjectArray=null,this.options={onDetected:t.onDetected||(()=>{}),onClosed:t.onClosed||(()=>{}),interval:t.interval||1e3,enableDebuggerDetection:t.enableDebuggerDetection||!1},!j.UA.Firefox)try{let e=new Blob([Do.workerScript],{type:"application/javascript"}),i=URL.createObjectURL(e),o=new Worker(i);this.workerConsole=new Do(o,i)}catch(e){console.warn("Failed to create Web Worker for DevTools detection:",e)}this.startDetection()}createLargeObjectArray(){let t={};for(let i=0;i<500;i++)t[`${i}`]=`${i}`;let e=[];for(let i=0;i<50;i++)e.push(t);return e}getLargeObjectArray(){return this.largeObjectArray===null&&(this.largeObjectArray=this.createLargeObjectArray()),this.largeObjectArray}async calcTablePrintTime(){let t=this.getLargeObjectArray();if(this.workerConsole)try{return(await this.workerConsole.table(t)).time}catch{let e=performance.now();return Io(t),performance.now()-e}else{let e=performance.now();return Io(t),performance.now()-e}}async calcLogPrintTime(){let t=this.getLargeObjectArray();if(this.workerConsole)return(await this.workerConsole.log(t)).time;{let e=performance.now();return Pr(t),performance.now()-e}}isPerformanceDetectionEnabled(){return xr({includes:[()=>!!j.UA.Chrome,()=>!!j.UA.Chromium,()=>!!j.UA.Safari,()=>!!j.UA.Firefox],excludes:[]})}isDebuggerDetectionEnabled(){return this.options.enableDebuggerDetection}async checkPerformanceBased(){if(!this.isPerformanceDetectionEnabled())return!1;let t=await this.calcTablePrintTime(),e=Math.max(await this.calcLogPrintTime(),await this.calcLogPrintTime());return this.maxPrintTime=Math.max(this.maxPrintTime,e),this.workerConsole?await this.workerConsole.clear():Fr(),t===0?!1:this.maxPrintTime===0?!!await Ho():t>this.maxPrintTime*10}async checkDebuggerBased(){if(!this.isDebuggerDetectionEnabled()||await Ho())return!1;let t=performance.now();try{(()=>{}).constructor("debugger")()}catch{debugger}return performance.now()-t>100}async detectDevTools(){return await this.checkPerformanceBased()?!0:this.options.enableDebuggerDetection&&this.checkCount>=this.maxChecks?await this.checkDebuggerBased():!1}startDetection(){this.intervalId=window.setInterval(async()=>{this.checkCount++;let t=await this.detectDevTools();t!==this.isOpen&&(this.isOpen=t,t?this.options.onDetected():this.options.onClosed()),this.checkCount>this.maxChecks*2&&(this.checkCount=0)},this.options.interval),window.addEventListener("beforeunload",()=>this.destroy())}isDevToolsOpen(){return this.isOpen}async checkNow(){let t=this.isOpen;return this.isOpen=await this.detectDevTools(),this.isOpen!==t&&(this.isOpen?this.options.onDetected():this.options.onClosed()),this.isOpen}destroy(){this.intervalId&&(clearInterval(this.intervalId),this.intervalId=void 0),this.workerConsole&&(this.workerConsole.destroy(),this.workerConsole=void 0),this.isOpen=!1,this.checkCount=0}};var fi=class{constructor(t){if(this.detected=!1,!t.onDetected)throw new Error("onDetected callback is required");this.options=t,this.setupDetection()}isIframed(){try{return window.self!==window.top?{isEmbedded:!0,isCrossOrigin:!window.top.location.href}:{isEmbedded:!1,isCrossOrigin:!1}}catch{return{isEmbedded:!0,isCrossOrigin:!0}}}setupDetection(){let{isEmbedded:t,isCrossOrigin:e}=this.isIframed();if(t){this.handleDetected(e);return}this.observer=new MutationObserver(()=>{let{isEmbedded:i,isCrossOrigin:o}=this.isIframed();i&&!this.detected&&(this.handleDetected(o),this.observer?.disconnect())}),this.observer.observe(document.documentElement,{childList:!0,subtree:!0,attributes:!0}),window.addEventListener("unload",()=>this.destroy())}handleDetected(t){this.detected=!0,this.options.onDetected?.(t)}destroy(){this.observer?.disconnect(),this.observer=void 0,this.detected=!1}};var Si=class{constructor(t={}){this.styleElement=null,this.beforePrintHandler=null,this.onPrintAttempt=t.onPrintAttempt,t.disable&&this.setupPrintProtection(t.watermark)}setupPrintProtection(t){let e=document.createElement("style");e.textContent=`
            @media print {
                body * {
                    display: none !important;
                }
                body::after {
                    content: "${t||"Printing has been disabled"}";
                    font-size: 200%;
                    display: block;
                    text-align: center;
                    margin-top: 50vh;
                    transform: translateY(-50%);
                }
            }
        `,document.head.appendChild(e),this.styleElement=e,this.beforePrintHandler=i=>(i.preventDefault(),this.onPrintAttempt?.(),!1),window.addEventListener("beforeprint",this.beforePrintHandler)}destroy(){this.beforePrintHandler&&(window.removeEventListener("beforeprint",this.beforePrintHandler),this.beforePrintHandler=null),this.styleElement?.parentNode&&(this.styleElement.parentNode.removeChild(this.styleElement),this.styleElement=null)}};var yi=class{constructor(t={}){this.onContextMenuBlocked=t.onContextMenuBlocked,this.contextMenuHandler=this.handleContextMenu.bind(this),document.addEventListener("contextmenu",this.contextMenuHandler,!0),window.addEventListener("unload",()=>this.destroy())}handleContextMenu(t){t.preventDefault(),t.stopPropagation();let e={type:"context_menu",timestamp:Date.now(),clientX:t.clientX,clientY:t.clientY,targetFrameSrc:""};return this.onContextMenuBlocked&&this.onContextMenuBlocked(e),!1}destroy(){this.contextMenuHandler&&(document.removeEventListener("contextmenu",this.contextMenuHandler,!0),this.contextMenuHandler=void 0)}};var bi="readium:navigator:suspiciousActivity",_i=class{dispatchSuspiciousActivity(t,e){let i=new CustomEvent(bi,{detail:{type:t,timestamp:Date.now(),...e}});window.dispatchEvent(i)}constructor(t={}){t.monitorDevTools&&(this.devToolsDetector=new gi({onDetected:()=>{this.dispatchSuspiciousActivity("developer_tools",{targetFrameSrc:"",key:"",code:"",keyCode:-1,ctrlKey:!1,altKey:!1,shiftKey:!1,metaKey:!1})}})),t.checkAutomation&&(this.automationDetector=new pi({onDetected:e=>{this.dispatchSuspiciousActivity("automation_detected",{tool:e})}})),t.checkIFrameEmbedding&&(this.iframeEmbeddingDetector=new fi({onDetected:e=>{this.dispatchSuspiciousActivity("iframe_embedding_detected",{isCrossOrigin:e})}})),t.protectPrinting?.disable&&(this.printProtector=new Si({...t.protectPrinting,onPrintAttempt:()=>{this.dispatchSuspiciousActivity("print",{})}})),t.disableContextMenu&&(this.contextMenuProtector=new yi({onContextMenuBlocked:e=>{this.dispatchSuspiciousActivity("context_menu",e)}}))}destroy(){this.automationDetector?.destroy(),this.devToolsDetector?.destroy(),this.iframeEmbeddingDetector?.destroy(),this.printProtector?.destroy(),this.contextMenuProtector?.destroy()}};var Ri="readium:navigator:keyboardPeripheral",vi=class{constructor(t={}){this.keyManager=new Ht,this.pagehideHandler=e=>{e.persisted||this.destroy()},this.dispatcher=e=>{let i=new CustomEvent(Ri,{detail:e});window.dispatchEvent(i)},this.conditionBridge=new Pt(t.keyboardPeripherals||[],e=>this.setupKeyboardPeripherals(e)),this.conditionBridge.setup(),window.addEventListener("pagehide",this.pagehideHandler)}setupKeyboardPeripherals(t){this.removeKeydownHandler(),t.length!==0&&(this.keydownHandler=this.keyManager.createUnifiedHandler("",t,this.dispatcher),document.addEventListener("keydown",this.keydownHandler,!0))}removeKeydownHandler(){this.keydownHandler&&(document.removeEventListener("keydown",this.keydownHandler,!0),this.keydownHandler=void 0)}destroy(){this.conditionBridge.destroy(),this.removeKeydownHandler(),window.removeEventListener("pagehide",this.pagehideHandler)}};var kr=r=>r.split(/[?#]/)[0],Pa=r=>{let t=kr(r);if(t.endsWith("/"))return t;try{let e=new URL(t);return e.pathname=e.pathname.substring(0,e.pathname.lastIndexOf("/")+1),e.href}catch{return t.replace(/[^/]*$/,"")}},Fa=(r,t="always",e)=>{let i=t==="always"||(t==="never"?!1:typeof e=="string"&&e.startsWith("https:")),o=r.join(" ");return[...i?["upgrade-insecure-requests"]:[],`default-src ${o} blob:`,"connect-src 'none'",`script-src ${o} blob: 'unsafe-inline'`,`style-src ${o} blob: 'unsafe-inline'`,`img-src ${o} blob: data:`,`font-src ${o} blob: data:`,`object-src ${o} blob:`,`child-src ${o}`,"form-action 'none'"].join("; ")},ue=class{constructor(t,e,i,o){this.pub=t,this.baseURL=e,this.item=i,this.injector=null,this.item=i,this.cssProperties=o.cssProperties,this.injector=o.injector??null,this.upgradeInsecureRequests=o.upgradeInsecureRequests??"always"}reset(){this.currentUrl&&URL.revokeObjectURL(this.currentUrl),this.currentUrl=void 0,this.currentResource?.close(),this.currentResource=void 0,this.pendingBuild=void 0}async build(t=!1){if(this.currentUrl)return this.currentUrl;if(this.pendingBuild)return this.pendingBuild;let e=this.doBuild(t).finally(()=>{this.pendingBuild===e&&(this.pendingBuild=void 0)});return this.pendingBuild=e,e}async doBuild(t){this.currentResource=this.pub.get(this.item);let e=await this.currentResource.link();if(!this.currentResource)return"about:blank";if(e.mediaType.isHTML){let i=await this.buildHtmlFrame(t);return this.currentUrl=i,i}else if(e.mediaType.isBitmap||e.mediaType.equals(Y.SVG)){let i=await this.buildImageFrame();return this.currentUrl=i,i}else throw Error("Unsupported frame mediatype "+e.mediaType.string)}async buildHtmlFrame(t=!1){if(!this.currentResource)throw new Error("No resource loaded");let e=await this.currentResource.link(),i=await this.currentResource.readAsXML();if(!i)throw new Error(`Failed reading item ${e.href}`);let o=i.querySelector("parsererror");if(o){let n=o.querySelector("div");throw new Error(`Failed parsing item ${e.href}: ${n?.textContent||o.textContent}`)}return this.injector&&await this.injector.injectForDocument(i,e),this.finalizeDOM(i,this.pub.baseURL,e.toURL(this.baseURL)||"",e.mediaType,t,this.cssProperties,this.upgradeInsecureRequests)}async buildImageFrame(){if(!this.currentResource)throw new Error("No resource loaded");let t=await this.currentResource.link(),e=t.toURL(this.baseURL)||"",i=document.implementation.createHTMLDocument(t.title||t.href);if((t?.height||0)>0&&(t?.width||0)>0){let s=i.createElement("meta");s.name="viewport",s.content=`width=${t.width}, height=${t.height}`,s.dataset.readium="true",i.head.appendChild(s)}let o=document.createElement("img");o.src=e||"",o.alt=t.title||"",await o.decode(),i.body.appendChild(o),this.injector&&await this.injector.injectForDocument(i,new q({href:"readium-image-frame.xhtml",type:Y.XHTML.string}));let n=i.createElement("style");return n.dataset.readium="true",n.textContent=`
        html, body { width: 100%; height: 100%; margin: 0; padding: 0; font-size: 0; }
        img { margin: 0; padding: 0; border: 0; }`,i.head.appendChild(n),this.finalizeDOM(i,this.pub.baseURL,e,t.mediaType,!0,void 0,this.upgradeInsecureRequests)}setProperties(t,e){for(let i in t){let o=t[i];o&&e.documentElement.style.setProperty(i,o)}}finalizeDOM(t,e,i,o,n=!1,s,a="always"){if(!t)return"";let h=(this.injector?.getAllowedDomains?.()||[]).map(kr),l=e?Pa(e):void 0,d=[...new Set([...l?[l]:[],...h])].filter(Boolean);if(s&&!n&&this.setProperties(s,t),t.body.querySelectorAll("img").forEach(m=>{m.setAttribute("fetchpriority","high"),m.setAttribute("referrerpolicy","origin")}),o.isHTML&&this.pub.metadata.languages?.[0]){let m=this.pub.metadata.languages[0];if(o===Y.XHTML){let p=t.documentElement.lang||t.documentElement.getAttribute("xml:lang"),g=t.body.lang||t.body.getAttribute("xml:lang");g&&!p?(t.documentElement.lang=g,t.documentElement.setAttribute("xml:lang",g),t.body.removeAttribute("xml:lang"),t.body.removeAttribute("lang")):p||(t.documentElement.lang=m,t.documentElement.setAttribute("xml:lang",m))}else o===Y.HTML&&!t.documentElement.lang&&(t.documentElement.lang=m)}if(Ft(this.pub.metadata)==="rtl"&&!t.documentElement.dir&&!t.body.dir&&(t.documentElement.dir=E.rtl),i!==void 0){let m=t.createElement("base");m.href=i,m.dataset.readium="true",t.head.firstChild.before(m)}let c=t.createElement("meta");return c.httpEquiv="Content-Security-Policy",c.content=Fa(d,a,e),c.dataset.readium="true",t.head.firstChild.before(c),URL.createObjectURL(new Blob([new XMLSerializer().serializeToString(t)],{type:o.isHTML?o.string:"application/xhtml+xml"}))}};var gt=class{constructor(t,e={},i=[],o=[],n=[]){this.timelineFragmentIds=o,this.hidden=!0,this.destroyed=!1,this.pendingUnwatchSelectors=[],this.currModules=[],this.serialize=ai(),this.frame=document.createElement("iframe"),this.frame.sandbox.value="allow-same-origin allow-scripts",this.frame.classList.add("readium-navigator-iframe"),this.frame.style.visibility="hidden",this.frame.ariaHidden="true",this.frame.style.opacity="0",this.frame.style.position="absolute",this.frame.style.pointerEvents="none",this.frame.style.transition="visibility 0s, opacity 0.1s linear",this.source=t,this.contentProtectionConfig={...e},this.keyboardPeripheralsConfig=[...i],this.resizeWatchSelectors=[...n]}async load(t){return this.serialize(()=>new Promise((e,i)=>{if(this.loader){let o=this.frame.contentWindow;if([...this.currModules].sort().join("|")===[...t].sort().join("|")){try{e(o)}catch{}return}this.comms?.halt(),this.loader.destroy(),this.loader=new bt(o,t),this.currModules=t,this.comms=void 0;try{e(o)}catch{}return}this.frame.onload=()=>{let o=this.frame.contentWindow;this.loader=new bt(o,t),this.currModules=t;try{e(o)}catch{}},this.frame.onerror=o=>{try{i(o)}catch{}},this.frame.contentWindow.location.replace(this.source)}))}applyContentProtection(){this.comms||this.comms.resume(),this.comms.send("peripherals_protection",this.contentProtectionConfig),this.keyboardPeripheralsConfig&&this.keyboardPeripheralsConfig.length>0&&(this.conditionBridge?.destroy(),this.conditionBridge=new Pt(this.keyboardPeripheralsConfig,t=>{t.length>0&&this.comms.send("keyboard_peripherals",t)}),this.conditionBridge.setup()),this.contentProtectionConfig.monitorScrollingExperimental&&this.comms.send("scroll_protection",{}),this.contentProtectionConfig.protectPrinting?.disable&&this.comms.send("print_protection",this.contentProtectionConfig.protectPrinting),this.resizeWatchSelectors.forEach(t=>{this.comms.send("decoration_resize",{action:"watch",selector:t})}),this.pendingUnwatchSelectors.forEach(t=>{this.comms.send("decoration_resize",{action:"unwatch",selector:t})}),this.pendingUnwatchSelectors=[]}addResizeTarget(t){this.resizeWatchSelectors.includes(t)||this.resizeWatchSelectors.push(t),this.pendingUnwatchSelectors=this.pendingUnwatchSelectors.filter(e=>e!==t),this.comms?.ready&&this.comms.send("decoration_resize",{action:"watch",selector:t})}removeResizeTarget(t){this.resizeWatchSelectors=this.resizeWatchSelectors.filter(e=>e!==t),this.comms?.ready?this.comms.send("decoration_resize",{action:"unwatch",selector:t}):this.pendingUnwatchSelectors.includes(t)||this.pendingUnwatchSelectors.push(t)}async destroy(){return this.serialize(async()=>{this.conditionBridge?.destroy(),await this.hideNow(),this.loader?.destroy(),this.frame.remove(),this.destroyed=!0})}async hide(){return this.serialize(()=>this.hideNow())}async hideNow(){if(!this.destroyed){if(this.frame.style.visibility="hidden",this.frame.ariaHidden="true",this.frame.style.opacity="0",this.frame.style.pointerEvents="none",this.hidden=!0,this.frame.blur(),this.frame.parentElement)return this.comms===void 0||!this.comms.ready?void 0:new Promise((t,e)=>{this.comms?.send("unfocus",void 0,i=>{this.comms?.halt(),t()})});this.comms?.halt()}}async show(t){return this.serialize(()=>this.showNow(t))}async showNow(t){if(this.destroyed)throw Error("Trying to show frame when it doesn't exist");if(!this.frame.parentElement)throw Error("Trying to show frame that is not attached to the DOM");return this.comms?this.comms.resume():this.comms=new jt(this.frame.contentWindow,this.source),new Promise((e,i)=>{this.comms?.send("activate",void 0,()=>{this.comms?.send("focus",void 0,()=>{this.applyContentProtection(),this.timelineFragmentIds.length>0&&this.comms?.send("timeline_entries",this.timelineFragmentIds);let o=()=>{this.frame.style.removeProperty("visibility"),this.frame.style.removeProperty("opacity"),this.frame.style.removeProperty("pointer-events"),this.frame.ariaHidden=null,this.hidden=!1,j.UA.WebKit&&this.comms?.send("force_webkit_recalc",void 0),e()};t!==void 0?this.comms?.send("go_progression",t,o):o()})})})}setCSSProperties(t){this.destroyed||!this.frame.contentWindow||(this.hidden&&!this.comms&&(this.comms=new jt(this.frame.contentWindow,this.source),this.comms.halt()),this.comms?.send("update_properties",t))}get iframe(){if(this.destroyed)throw Error("Trying to use frame when it doesn't exist");return this.frame}get realSize(){if(this.destroyed)throw Error("Trying to use frame client rect when it doesn't exist");return this.frame.getBoundingClientRect()}get isDestroyed(){return this.destroyed}get window(){if(this.destroyed||!this.frame.contentWindow)throw Error("Trying to use frame window when it doesn't exist");return this.frame.contentWindow}get atLeft(){return this.window.scrollX<5}get atRight(){return this.window.scrollX>this.window.document.scrollingElement.scrollWidth-this.window.innerWidth-5}get msg(){return this.comms}get ldr(){return this.loader}};var Tr=10,zr=5,wi=class{constructor(t,e,i,o,n,s,a,h=[],l){this.pool=new Map,this.blobs=new Map,this.inprogress=new Map,this.pendingUpdates=new Map,this.injector=null,this.updateSequence=0,this.container=t,this.positions=e,this.currentCssProperties=i,this.injector=o??null,this.contentProtectionConfig=n||{},this.keyboardPeripheralsConfig=s||[],this.getFragmentIds=a??(()=>[]),this.resizeWatchSelectors=[...h],this.upgradeInsecureRequests=l??"always"}addResizeTarget(t){this.resizeWatchSelectors.includes(t)||this.resizeWatchSelectors.push(t),this.pool.forEach(e=>e.addResizeTarget(t))}removeResizeTarget(t){this.resizeWatchSelectors=this.resizeWatchSelectors.filter(e=>e!==t),this.pool.forEach(e=>e.removeResizeTarget(t))}async destroy(){let t=this.inprogress.values(),e=t.next(),i=[];for(;e.value;)i.push(e.value),e=t.next();i.length>0&&await Promise.allSettled(i),this.inprogress.clear();let o=this.pool.values(),n=o.next();for(;n.value;)await n.value.destroy(),n=o.next();this.pool.clear(),this.blobs.forEach(s=>s.reset()),this.blobs.clear(),this.injector?.dispose(),this.container.childNodes.forEach(s=>{(s.nodeType===Node.ELEMENT_NODE||s.nodeType===Node.TEXT_NODE)&&s.remove()})}async update(t,e,i,o=!1){for(;this.forcedUpdate;)await this.forcedUpdate;let n=++this.updateSequence,s=this.positions.findIndex(l=>l.locations.position===e.locations.position);if(s<0)throw Error(`Locator not found in position list: ${e.locations.position} > ${this.positions.reduce((l,d)=>d.locations.position||0>l?d.locations.position||0:l,0)}`);let a=this.positions[s].href;this.inprogress.has(a)&&await this.inprogress.get(a);let h=new Promise(async(l,d)=>{let c=[],m=[];this.positions.forEach((u,f)=>{(f>s+Tr||f<s-Tr)&&(c.includes(u.href)||c.push(u.href)),f<s+zr&&f>s-zr&&(m.includes(u.href)||m.push(u.href))}),c.forEach(async u=>{m.includes(u)||this.pool.has(u)&&(await this.pool.get(u)?.destroy(),this.pool.delete(u),this.pendingUpdates.has(u)&&this.pendingUpdates.set(u,{inPool:!1}))}),this.currentBaseURL!==void 0&&t.baseURL!==this.currentBaseURL&&(this.blobs.forEach(u=>u.reset()),this.blobs.clear()),this.currentBaseURL=t.baseURL,o&&(this.blobs.forEach(u=>u.reset()),this.blobs.clear(),this.pendingUpdates.clear());let p=async u=>{if(this.pendingUpdates.has(u)&&this.pendingUpdates.get(u)?.inPool===!1){let b=this.blobs.get(u);b&&(b.reset(),this.blobs.delete(u),this.pendingUpdates.delete(u))}if(this.pool.has(u)){let b=this.pool.get(u);if(!this.blobs.has(u))await b.destroy(),this.pool.delete(u),this.pendingUpdates.delete(u);else{await b.load(i);return}}let f=t.readingOrder.findWithHref(u);if(!f)return;this.blobs.has(u)||this.blobs.set(u,new ue(t,this.currentBaseURL||"",f,{cssProperties:this.currentCssProperties,injector:this.injector,upgradeInsecureRequests:this.upgradeInsecureRequests}));let S=new gt(await this.blobs.get(u).build(),this.contentProtectionConfig,this.keyboardPeripheralsConfig,this.getFragmentIds(u),this.resizeWatchSelectors);return u!==a&&await S.hide(),this.container.appendChild(S.iframe),await S.load(i),this.pool.set(u,S),S};try{await Promise.all(m.filter(u=>u===a).map(u=>p(u)))}catch(u){d(u);return}if(n!==this.updateSequence){l();return}Promise.all(m.filter(u=>u!==a&&!this.inprogress.has(u)).map(async u=>{let f=p(u);this.inprogress.set(u,f);try{await f}finally{this.inprogress.get(u)===f&&this.inprogress.delete(u)}})).catch(u=>console.warn("Failed preloading frame",u));let g=this.pool.get(a);if(!g){l();return}if(g?.source!==this._currentFrame?.source||o){if(await this._currentFrame?.hide(),n!==this.updateSequence){l();return}let u=this.pool.get(a);if(!u){l();return}if(await u.load(i),n!==this.updateSequence){l();return}let f=this.pool.get(a);if(!f){l();return}if(await f.show(e.locations.progression),this._currentFrame=f,g){let S=this.container.ownerDocument.activeElement;S&&S.tagName==="IFRAME"&&S!==g.iframe&&g.iframe.focus({preventScroll:!0})}}l()});if(this.inprogress.set(a,h),o){let l=h.catch(()=>{});this.forcedUpdate=l,l.then(()=>{this.forcedUpdate===l&&(this.forcedUpdate=void 0)})}try{await h}finally{this.inprogress.delete(a)}}setCSSProperties(t){if(!((e,i)=>{let o=Object.keys(e),n=Object.keys(i);if(o.length!==n.length)return!1;for(let s of o)if(e[s]!==i[s])return!1;return!0})(this.currentCssProperties||{},t)){this.currentCssProperties=t,this.pool.forEach(e=>{e.setCSSProperties(t)});for(let e of this.blobs.keys())this.pendingUpdates.set(e,{inPool:this.pool.has(e)})}}get currentFrames(){return[this._currentFrame]}_frameFor(t){return this.pool.get(t)}get currentBounds(){let t={x:0,y:0,width:0,height:0,top:0,right:0,bottom:0,left:0,toJSON(){return this}};return this.currentFrames.forEach(e=>{if(!e)return;let i=e.realSize;t.x=Math.min(t.x,i.x),t.y=Math.min(t.y,i.y),t.width+=i.width,t.height=Math.max(t.height,i.height),t.top=Math.min(t.top,i.top),t.right=Math.min(t.right,i.right),t.bottom=Math.min(t.bottom,i.bottom),t.left=Math.min(t.left,i.left)}),t}};var xa=(r=>(r[r.Left=0]="Left",r[r.Center=1]="Center",r[r.Right=2]="Right",r))(xa||{}),ka=(r=>(r[r.Top=0]="Top",r[r.Middle=1]="Middle",r[r.Bottom=2]="Bottom",r))(ka||{}),Ci=class{constructor(){this.outerWidth=0,this.outerHeight=0,this.HTML=document.documentElement,this.Head=document.head,this.Body=document.body}refreshOuterPixels(t){j.OS.iOS||(this.outerHeight=window.outerHeight-window.innerHeight,j.OS.Android&&j.UA.Chrome&&window.screen.height>window.innerHeight&&(this.outerHeight=(window.screen.height-window.innerHeight)/1.5),this.outerWidth=window.outerWidth-window.innerWidth)}getBibiEventCoord(t,e=0){let i={X:0,Y:0};return/^touch/.test(t.type)?(i.X=t.touches[e].screenX,i.Y=t.touches[e].screenY):(i.X=t.screenX,i.Y=t.screenY),(t.target.ownerDocument?.documentElement||t.target.documentElement)===this.HTML&&(i.X-=this.HTML.scrollLeft+this.Body.scrollLeft,i.Y-=this.HTML.scrollTop+this.Body.scrollTop),i.X-=this.outerWidth,i.Y-=this.outerHeight,i}getTouchDistance(t){if(t.touches.length!==2)return 0;let e=t.touches[0].screenX-this.outerWidth,i=t.touches[0].screenY-this.outerHeight,o=t.touches[1].screenX-this.outerWidth,n=t.touches[1].screenY-this.outerHeight;return Math.sqrt(Math.pow(o-e,2)+Math.pow(n-i,2))}getTouchCenter(t){if(t.touches.length!==2)return null;let e=this.HTML.scrollLeft+this.Body.scrollLeft,i=this.HTML.scrollTop+this.Body.scrollTop,o=t.touches[0].screenX-this.outerWidth-e,n=t.touches[0].screenY-this.outerHeight-i,s=t.touches[1].screenX-this.outerWidth-e,a=t.touches[1].screenY-this.outerHeight-i;return{X:(o+s)/2,Y:(n+a)/2}}getBibiEvent(t){if(!t)return{Coord:null,Division:null,Ratio:null,Target:null};let e=this.getBibiEventCoord(t),i=.3,o={X:e.X/window.innerWidth,Y:e.Y/window.innerHeight},n,s,a,h;a=n=i,h=s=1-i;let l={X:null,Y:null};return o.X<a?l.X=0:h<o.X?l.X=2:l.X=1,o.Y<n?l.Y=0:s<o.Y?l.Y=2:l.Y=1,{Target:t.target,Coord:e,Ratio:o,Division:l}}};var Ei=class{constructor(){this._DOM={show:!1,pinchTarget:document.createElement("div"),touch1:document.createElement("div"),touch2:document.createElement("div"),center:document.createElement("div"),stats:document.createElement("div")},this._DOM.show=!0,this._DOM.pinchTarget.style.zIndex=this._DOM.stats.style.zIndex=this._DOM.center.style.zIndex=this._DOM.touch1.style.zIndex=this._DOM.touch2.style.zIndex="100000",this._DOM.pinchTarget.style.position=this._DOM.stats.style.position=this._DOM.center.style.position=this._DOM.touch1.style.position=this._DOM.touch2.style.position="absolute",this._DOM.pinchTarget.style.borderRadius=this._DOM.center.style.borderRadius=this._DOM.touch1.style.borderRadius=this._DOM.touch2.style.borderRadius="50%",this._DOM.pinchTarget.style.pointerEvents=this._DOM.stats.style.pointerEvents=this._DOM.center.style.pointerEvents=this._DOM.touch1.style.pointerEvents=this._DOM.touch2.style.pointerEvents="none",this._DOM.pinchTarget.style.display=this._DOM.center.style.display=this._DOM.touch1.style.display=this._DOM.touch2.style.display="none",this._DOM.pinchTarget.style.paddingTop=this._DOM.center.style.paddingTop="10px",this._DOM.pinchTarget.style.width=this._DOM.pinchTarget.style.height=this._DOM.center.style.width=this._DOM.center.style.height="10px",this._DOM.pinchTarget.style.backgroundColor="green",this._DOM.center.style.backgroundColor="red",this._DOM.touch1.style.backgroundColor=this._DOM.touch2.style.backgroundColor="blue",this._DOM.touch1.style.height=this._DOM.touch2.style.height="20px",this._DOM.touch1.style.width=this._DOM.touch2.style.width="20px",this._DOM.touch1.style.paddingTop=this._DOM.touch2.style.paddingTop="20px",this._DOM.touch1.textContent="1",this._DOM.touch2.textContent="2",this._DOM.stats.style.padding="20px",this._DOM.stats.style.backgroundColor="rgba(0,0,0,0.5)",this._DOM.stats.style.color="white",this._DOM.stats.textContent="[stats]",document.body.appendChild(this._DOM.stats),document.body.appendChild(this._DOM.center),document.body.appendChild(this._DOM.touch1),document.body.appendChild(this._DOM.touch2),document.body.appendChild(this._DOM.pinchTarget)}get show(){return this.DOM.show}get DOM(){return this._DOM}};var Lr=6,Wo=1.02,Or=50,mt=()=>typeof window.CSSTransformValue<"u",Pi=class{constructor(t,e=!1){this.dragState=0,this.minimumMoved=!1,this.pan={startX:0,endX:0,startY:0,overscrollX:0,overscrollY:0,letItGo:!1,preventClick:!1,translateX:0,translateY:0,touchID:0},this.pinch={startDistance:0,startScale:0,target:{X:0,Y:0},touchN:0,startTranslate:{X:0,Y:0}},this._scale=1,this.scaleDebouncer=0,this.frameBounds=null,this.debugger=null,this.btouchstartHandler=this.touchstartHandler.bind(this),this.btouchendHandler=this.touchendHandler.bind(this),this.btouchmoveHandler=this.touchmoveHandler.bind(this),this.bdblclickHandler=this.dblclickHandler.bind(this),this.bmousedownHandler=this.mousedownHandler.bind(this),this.bmouseupHandler=this.mouseupHandler.bind(this),this.bmousemoveHandler=this.mousemoveHandler.bind(this),this.moveFrame=0,this.dragTranslate=null,this.dragTransform=null,this.manager=t,this.coordinator=new Ci,this.attachEvents(),e&&(this.debugger=new Ei)}get scale(){return this._scale}set scale(t){isNaN(t)&&(t=1),window.clearTimeout(this.scaleDebouncer),this.scaleDebouncer=window.setTimeout(()=>{this.dragState===0&&this.scale<Wo&&(this.pan.translateX=0,this.pan.translateY=0,this.clearPan(),this.manager.updateBookStyle()),this.manager.listener("zoom",t)},100),this._scale=t}attachEvents(){this.observe(this.manager.spineElement),this.pan={startX:0,startY:0,endX:0,overscrollX:0,overscrollY:0,letItGo:!1,preventClick:!1,translateX:0,translateY:0,touchID:0},this.pinch={startDistance:0,startScale:0,target:{X:0,Y:0},startTranslate:{X:0,Y:0},touchN:0}}clearPan(){this.pan.letItGo=!1,this.pan.touchID=0,this.pan.endX=0,this.pan.overscrollX=0,this.pan.overscrollY=0}clearPinch(){this.pinch={startDistance:0,startScale:this.pinch.startScale,target:{X:0,Y:0},touchN:0,startTranslate:{X:0,Y:0}}}observe(t){t.addEventListener("touchstart",this.btouchstartHandler),t.addEventListener("touchend",this.btouchendHandler),t.addEventListener("touchmove",this.btouchmoveHandler,{passive:!0}),t.addEventListener("dblclick",this.bdblclickHandler,{passive:!0}),t.addEventListener("mousedown",this.bmousedownHandler),t.addEventListener("mouseup",this.bmouseupHandler),t.addEventListener("mousemove",this.bmousemoveHandler)}clickHandler(t){}touchstartHandler(t){if(["TEXTAREA","OPTION","INPUT","SELECT"].indexOf(t.target.nodeName)!==-1)return;switch(t.stopPropagation(),this.frameBounds=this.manager.currentBounds,this.coordinator.refreshOuterPixels(this.frameBounds),t.touches.length){case 3:return;case 2:{t.preventDefault(),this.pinch.startDistance=this.coordinator.getTouchDistance(t);let i=this.startTouch(t);this.pan.startX=i.X,this.pan.startY=i.Y,this.dragState=2,this.manager.updateBookStyle(!0),this.isScaled?(this.pinch.target.X-=this.pan.translateX*(this.pinch.startScale/this.scale),this.pinch.target.Y-=this.pan.translateY*(this.pinch.startScale/this.scale),this.pinch.target={X:0,Y:0},this.pinch.startScale=1/this.scale):(this.pinch.target={X:0,Y:0},this.pinch.startScale=this.scale),this.pinch.startTranslate={X:this.pan.translateX,Y:this.pan.translateY},this.debugger?.show&&(this.debugger.DOM.touch2.style.display="",this.debugger.DOM.center.style.display="",this.debugger.DOM.pinchTarget.style.display="");return}case 1:this.pan.touchID=t.touches[0].identifier,this.debugger?.show&&(this.debugger.DOM.touch1.style.display="");default:this.dragState<1&&(this.dragState=1),this.manager.updateBookStyle(!0)}this.manager.updateSpineStyle(!1);let e=this.startTouch(t);this.pan.startX=e.X,this.pan.startY=e.Y}startTouch(t){let e=this.coordinator.getTouchCenter(t)||this.coordinator.getBibiEventCoord(t);return{X:e.X-this.manager.width/2-this.pan.translateX*this.scale+this.manager.width/2,Y:e.Y-this.manager.height/2-this.pan.translateY*this.scale+this.manager.height/2}}touchendHandler(t){if(t.stopPropagation(),!t.touches||t.touches.length===0)this.pan.endX&&!this.isScaled?(this.pinch.touchN&&(this.pan.endX=this.pan.startX),this.updateAfterDrag()):!this.pinch.touchN&&Math.abs(this.pan.overscrollX)>Or&&Math.abs(this.pan.overscrollY)<Or/2&&(this.pan.startX=0,this.pan.endX=-this.pan.overscrollX,this.updateAfterDrag()),this.dragState=0,this.minimumMoved=!1,this.clearPinch(),this.debugger?.show&&(this.debugger.DOM.center.style.display="none",this.debugger.DOM.touch1.style.display="none",this.debugger.DOM.touch2.style.display="none");else if(t.touches.length===1){this.dragState=1,t.touches[0].identifier!==this.pan.touchID&&(this.pan.touchID=t.touches[0].identifier),this.debugger?.show&&(this.debugger.DOM.center.style.display="none",this.debugger.DOM.touch2.style.display="none",this.debugger.DOM.pinchTarget.style.display="none");let e=this.startTouch(t);this.pan.startX=e.X,this.pan.startY=e.Y}window.setTimeout(()=>{this.manager.updateBookStyle(!0),this.dragState===0&&(this.scale<Wo&&(this.pan.translateX=0,this.pan.translateY=0),this.clearPan()),this.manager.updateBookStyle(!0)},50)}touchmoveHandler(t){t.stopPropagation();let e=this.coordinator.getBibiEventCoord(t);Math.abs(this.pan.startY-e.Y)+Math.abs(this.pan.startX-e.X)>5&&(this.minimumMoved||(this.manager.deselect(),this.minimumMoved=!0),this.dragState<1&&(this.dragState=1));let i=this.coordinator?.getTouchDistance(t),o=!1,n=this.scale;if(this.dragState===2&&i){if(this.pinch.touchN++,this.pinch.touchN<4)return;let s=i/this.pinch.startDistance*this.scale;s>=Lr&&(s=Lr),s<=Wo&&(s=1),this.scale=s,this.pinch.startDistance=i,o=!0}if(this.pan.letItGo===!1&&(this.pan.letItGo=Math.abs(this.pan.startY-e.Y)<Math.abs(this.pan.startX-e.X)),this.debugger?.show&&(this.debugger.DOM.touch1.style.top=`${e.Y-10}px`,this.debugger.DOM.touch1.style.left=`${e.X-10}px`,this.debugger.DOM.touch1.innerText=`${e.X.toFixed(2)},${e.Y.toFixed(2)}`),this.dragState>0&&this.isScaled||this.dragState>1){if(this.dragState===1){let h={X:e.X-this.manager.width/2,Y:e.Y-this.manager.height/2};this.pan.translateX=(h.X-(this.pan.startX-this.manager.width/2))*1/this.scale,this.pan.translateY=(h.Y-(this.pan.startY-this.manager.height/2))*1/this.scale}else if(this.dragState===2){let h=this.coordinator.getTouchCenter(t);if(this.debugger?.show){this.debugger.DOM.center.style.top=`${h.Y-5}px`,this.debugger.DOM.center.style.left=`${h.X-5}px`,this.debugger.DOM.center.innerText=`${h.X.toFixed(2)},${h.Y.toFixed(2)}`;let p=this.coordinator.getBibiEventCoord(t,1);this.debugger.DOM.touch2.style.top=`${p.Y-10}px`,this.debugger.DOM.touch2.style.left=`${p.X-10}px`,this.debugger.DOM.touch2.innerText=`${p.X.toFixed(2)},${p.Y.toFixed(2)}`}h.X-=this.manager.width/2,h.Y-=this.manager.height/2;let l=-h.X/n;l+=h.X/this.scale,this.pinch.target.X+=l,h.X+=this.pinch.target.X*this.scale/this.pinch.startScale;let d=-h.Y/n;d+=h.Y/this.scale,this.pinch.target.Y+=d,h.Y+=this.pinch.target.Y*this.scale/this.pinch.startScale;let c=(h.X-(this.pan.startX-this.manager.width/2))*1/this.scale,m=(h.Y-(this.pan.startY-this.manager.height/2))*1/this.scale;this.pan.translateX=c,this.pan.translateY=m,this.debugger?.show&&(this.debugger.DOM.pinchTarget.style.left=`${this.pinch.target.X*this.scale/this.pinch.startScale-5+this.manager.width/2}px`,this.debugger.DOM.pinchTarget.style.top=`${this.pinch.target.Y*this.scale/this.pinch.startScale-5+this.manager.height/2}px`,this.debugger.DOM.pinchTarget.innerText=`${(this.pinch.target.X*this.scale/this.pinch.startScale).toFixed(2)},${(this.pinch.target.Y*this.scale/this.pinch.startScale).toFixed(2)}`)}let s=this.frameBounds.width/6,a=this.frameBounds.height/6;this.pan.translateX<-s&&(this.pan.overscrollX=-(s+this.pan.translateX),this.pan.translateX=-s),this.pan.translateY<-a&&(this.pan.overscrollY=-(a+this.pan.translateY),this.pan.translateY=-a),this.pan.translateX>s&&(this.pan.overscrollX=s-this.pan.translateX,this.pan.translateX=s),this.pan.translateY>a&&(this.pan.overscrollY=a-this.pan.translateY,this.pan.translateY=a),o=!0,this.debugger?.show&&(this.debugger.DOM.stats.innerText=`TX: ${this.pan.translateX.toFixed(2)}
TY: ${this.pan.translateY.toFixed(2)}
Zoom: ${this.scale.toFixed(2)}
Overscroll: ${this.pan.overscrollX.toFixed(2)},${this.pan.overscrollY.toFixed(2)}`)}if(o){this.manager.updateBookStyle();return}if(this.dragState>0&&this.pan.letItGo){this.pan.endX=e.X;let s=this.manager.currentSlide*(this.manager.width/this.manager.perPage),a=this.pan.endX-this.pan.startX,h=this.manager.rtl?s+a:s-a,l=window.devicePixelRatio||1,d=Math.round((this.manager.rtl?1:-1)*h*l)/l;cancelAnimationFrame(this.moveFrame),this.moveFrame=requestAnimationFrame(()=>{requestAnimationFrame(()=>{mt()?(this.dragTransform||(this.dragTranslate=new CSSTranslate(CSS.px(0),CSS.px(0),CSS.px(0)),this.dragTransform=new CSSTransformValue([this.dragTranslate])),this.dragTranslate.x.value=d,this.manager.spineElement.attributeStyleMap.set("transform",this.dragTransform)):this.manager.spineElement.style.transform=`translate3d(${d}px, 0, 0)`})})}}dblclickHandler(t){clearTimeout(this.dtimer),this.pdblclick=!0,this.dtimer=window.setTimeout(()=>this.pdblclick=!1,200),!this.disableDblClick&&this.isScaled&&(this.scale=1)}get isScaled(){return this.scale>1}addTouch(t){t.touches=[{pageX:t.pageX,pageY:t.pageY}]}mousedownHandler(t){this.isScaled&&(this.addTouch(t),this.touchstartHandler(t))}mouseupHandler(t){this.isScaled&&this.touchendHandler(t)}mousemoveHandler(t){this.isScaled&&t.buttons>0&&(t.preventDefault(),this.addTouch(t),this.touchmoveHandler(t))}updateAfterDrag(){let t=(this.manager.rtl?-1:1)*(this.pan.endX-this.pan.startX),e=Math.abs(t);t>0&&e>this.manager.threshold&&this.manager.slength>this.manager.perPage?this.manager.listener("no_less",void 0):t<0&&e>this.manager.threshold&&this.manager.slength>this.manager.perPage&&this.manager.listener("no_more",void 0),this.manager.slideToCurrent(!0,!0)}};var Fi=class{constructor(t,e,i,o={},n=[],s=[]){this.frameIsAppended=!1,this.pendingUnwatchSelectors=[],this.currModules=[],this.viewportSize=void 0,this.serialize=ai(),this.cachedPage=void 0,this.peripherals=t,this.debugHref=i,this.contentProtectionConfig={...o},this.keyboardPeripheralsConfig=[...n],this.resizeWatchSelectors=[...s],this.frame=document.createElement("iframe"),this.frame.sandbox.value="allow-same-origin allow-scripts",this.frame.classList.add("readium-navigator-iframe"),this.frame.classList.add("blank"),this.frame.scrolling="no",this.frame.style.visibility="hidden",this.frame.ariaHidden="true",this.frame.style.display="none",this.frame.style.position="absolute",this.frame.style.pointerEvents="none",this.frame.style.transformOrigin="0 0",this.frame.style.transform="scale(1)",this.frame.style.background="#fff",this.frame.style.touchAction="none",this.frame.dataset.originalHref=i,this.source="about:blank",this.wrapper=document.createElement("div"),this.wrapper.style.position="relative",this.wrapper.style.contain="strict",this.wrapper.style.float=this.wrapper.style.cssFloat=e===E.rtl?"right":"left"}async load(t,e){return this.serialize(()=>this.loadNow(t,e))}async loadNow(t,e){return this.frameIsAppended||(this.wrapper.appendChild(this.frame),this.frameIsAppended=!0),this.source===e&&this.loadPromise&&[...this.currModules].sort().join("|")===[...t].sort().join("|")?this.loadPromise:(this.loaded&&this.source!==e&&this.window.stop(),this.source=e,this.loadPromise=new Promise((i,o)=>{if(this.loader&&this.loaded){let n=this.frame.contentWindow;if([...this.currModules].sort().join("|")===[...t].sort().join("|")){try{i(n),this.loadPromise=void 0}catch{}return}this.comms?.halt(),this.loader.destroy(),this.loader=new bt(n,t),this.currModules=t,this.comms=void 0;try{i(n),this.loadPromise=void 0}catch{}return}this.frame.addEventListener("load",()=>{let n=this.frame.contentWindow;this.loader=new bt(n,t),this.currModules=t,this.peripherals.observe(this.wrapper),this.peripherals.observe(n);try{i(n)}catch{}},{once:!0}),this.frame.addEventListener("error",n=>{try{o(n.error),this.loadPromise=void 0}catch{}},{once:!0}),this.frame.style.removeProperty("display"),this.frame.contentWindow.location.replace(this.source)}),this.loadPromise)}loadPageSize(){if(this.viewportSize)return this.viewportSize;let t=this.frame.contentWindow,e=t.document.head.querySelector("meta[name=viewport]");if(e){let i=/(\w+) *= *([^\s,]+)/g,o,n=0,s=0;for(;o=i.exec(e.content);)o[1]==="width"?n=Number.parseFloat(o[2]):o[1]==="height"&&(s=Number.parseFloat(o[2]));if(n>0&&s>0)return this.viewportSize={width:n,height:s},this.viewportSize}return{width:t.document.body.scrollWidth,height:t.document.body.scrollHeight}}update(t){if(!this.loaded)return;let e=this.loadPageSize(),i=Math.min(this.wrapper.clientWidth/e.width,this.wrapper.clientHeight/e.height);mt()?(this.frame.attributeStyleMap.set("width",CSS.px(e.width)),this.frame.attributeStyleMap.set("height",CSS.px(e.height)),this.frame.attributeStyleMap.set("transform",new CSSTransformValue([new CSSScale(i,i)]))):(this.frame.style.height=`${e.height}px`,this.frame.style.width=`${e.width}px`,this.frame.style.transform=`scale(${i})`);let o=this.frame.getBoundingClientRect(),n=this.wrapper.clientHeight-o.height;if(mt())if(this.frame.attributeStyleMap.set("top",CSS.px(n/2)),t===I.left){let s=this.wrapper.clientWidth-o.width;this.frame.attributeStyleMap.set("left",CSS.px(s))}else if(t===I.center){let s=this.wrapper.clientWidth-o.width;this.frame.attributeStyleMap.set("left",CSS.px(s/2))}else this.frame.attributeStyleMap.set("left",CSS.px(0));else if(this.frame.style.top=`${n/2}px`,t===I.left){let s=this.wrapper.clientWidth-o.width;this.frame.style.left=`${s}px`}else if(t===I.center){let s=this.wrapper.clientWidth-o.width;this.frame.style.left=`${s/2}px`}else this.frame.style.left="0px";this.frame.style.removeProperty("visibility"),this.frame.ariaHidden=null,this.frame.style.removeProperty("pointer-events"),this.frame.classList.remove("blank"),this.frame.classList.add("loaded")}async destroy(){return this.serialize(async()=>{this.conditionBridge?.destroy(),await this.unfocusNow(),this.loader?.destroy(),this.wrapper.remove()})}async unload(){return this.serialize(()=>this.unloadNow())}async unloadNow(){if(this.loaded)return this.deselect(),this.frame.style.visibility="hidden",this.frame.ariaHidden="true",this.frame.style.pointerEvents="none",this.frame.classList.add("blank"),this.frame.classList.remove("loaded"),this.comms?.halt(),this.loader?.destroy(),this.comms=void 0,this.frame.blur(),new Promise((t,e)=>{this.frame.addEventListener("load",()=>{try{this.showPromise=void 0,t()}catch{}},{once:!0}),this.frame.addEventListener("error",i=>{try{this.showPromise=void 0,e(i.error)}catch{}},{once:!0}),this.source="about:blank",this.frame.contentWindow.location.replace("about:blank"),this.frame.style.display="none",this.frame.style.width="0px",this.frame.style.height="0px"})}deselect(){this.frame.contentWindow?.getSelection()?.removeAllRanges()}async unfocus(){return this.serialize(()=>this.unfocusNow())}async unfocusNow(){if(this.frame.parentElement)return this.comms===void 0||!this.comms.ready?void 0:(this.frame.blur(),new Promise((t,e)=>{this.comms?.send("unfocus",void 0,i=>{this.comms?.halt(),this.showPromise=void 0,t()})}));this.comms?.halt()}applyContentProtection(){this.comms||this.comms.resume(),this.comms.send("peripherals_protection",this.contentProtectionConfig),this.keyboardPeripheralsConfig&&this.keyboardPeripheralsConfig.length>0&&(this.conditionBridge?.destroy(),this.conditionBridge=new Pt(this.keyboardPeripheralsConfig,t=>{t.length>0&&this.comms.send("keyboard_peripherals",t)}),this.conditionBridge.setup()),this.contentProtectionConfig.protectPrinting?.disable&&this.comms.send("print_protection",this.contentProtectionConfig.protectPrinting),this.resizeWatchSelectors.forEach(t=>{this.comms.send("decoration_resize",{action:"watch",selector:t})}),this.pendingUnwatchSelectors.forEach(t=>{this.comms.send("decoration_resize",{action:"unwatch",selector:t})}),this.pendingUnwatchSelectors=[]}addResizeTarget(t){this.resizeWatchSelectors.includes(t)||this.resizeWatchSelectors.push(t),this.pendingUnwatchSelectors=this.pendingUnwatchSelectors.filter(e=>e!==t),this.comms?.ready&&this.comms.send("decoration_resize",{action:"watch",selector:t})}removeResizeTarget(t){this.resizeWatchSelectors=this.resizeWatchSelectors.filter(e=>e!==t),this.comms?.ready?this.comms.send("decoration_resize",{action:"unwatch",selector:t}):this.pendingUnwatchSelectors.includes(t)||this.pendingUnwatchSelectors.push(t)}async show(t){return this.serialize(()=>this.showNow(t))}async showNow(t){if(!this.frame.parentElement){console.warn("Trying to show frame that is not attached to the DOM");return}if(!this.loaded){this.showPromise=void 0;return}return this.showPromise?(this.cachedPage!==t&&(this.update(t),this.cachedPage=t),this.showPromise):(this.cachedPage=t,this.comms?this.comms.resume():this.comms=new jt(this.frame.contentWindow,this.source),this.showPromise=new Promise((e,i)=>{this.comms.send("focus",void 0,o=>{this.update(this.cachedPage),this.applyContentProtection(),e()})}),this.showPromise)}async activate(){return this.serialize(()=>new Promise((t,e)=>{if(!this.comms?.ready)return t();this.comms?.send("activate",void 0,()=>{t()})}))}get element(){return this.wrapper}get iframe(){return this.frame}get realSize(){return this.frame.getBoundingClientRect()}get loaded(){return this.frame.contentWindow&&this.frame.contentWindow.location.href!=="about:blank"}set width(t){if(mt()){let e=this.wrapper.attributeStyleMap.get("width");if(e instanceof CSSUnitValue&&e.unit==="percent"&&e.value===t)return;this.wrapper.attributeStyleMap.set("width",CSS.percent(t))}else{let e=`${t}%`;if(this.wrapper.style.width===e)return;this.wrapper.style.width=e}}set height(t){if(mt()){let e=this.wrapper.attributeStyleMap.get("height");if(e instanceof CSSUnitValue&&e.unit==="px"&&e.value===t)return;this.wrapper.attributeStyleMap.set("height",CSS.px(t))}else{let e=`${t}px`;if(this.wrapper.style.height===e)return;this.wrapper.style.height=e}}get window(){if(!this.frame.contentWindow)throw Error("Trying to use frame window when it doesn't exist");return this.frame.contentWindow}get atLeft(){return this.window.scrollX<5}get atRight(){return this.window.scrollX>this.window.document.scrollingElement.scrollWidth-this.window.innerWidth-5}get msg(){return this.comms}get ldr(){return this.loader}};var ki=(r=>(r.auto="auto",r.landscape="landscape",r.portrait="portrait",r))(ki||{}),Ti=(r=>(r.auto="auto",r.both="both",r.none="none",r.landscape="landscape",r))(Ti||{}),xi=class{constructor(t){this.shift=!0,this.spreads=[],this.nLandscape=0,this.index(t),this.testShift(t),console.log(`Indexed ${this.spreads.length} spreads for ${t.readingOrder.items.length} items`)}index(t,e=!1){this.nLandscape=0,t.readingOrder.items.forEach((i,o)=>{e||(t.readingOrder.items[o]=i.addProperties({number:o+1,isImage:i.type?.indexOf("image/")===0}));let n=i.properties?.otherProperties.orientation==="landscape";(!i.properties?.page||e)&&(i.properties=i.properties?.add({page:n?"center":((this.shift?0:1)+o-this.nLandscape)%2?t.metadata.readingProgression===E.rtl?"right":"left":t.metadata.readingProgression===E.rtl?"left":"right"})),(n||i.properties?.otherProperties.addBlank)&&this.nLandscape++}),e&&(this.spreads=[]),this.buildSpreads(t.readingOrder)}testShift(t){let e=!1;this.spreads.forEach((i,o)=>{if(i.length>1)return;let n=i[0],s=n.properties?.otherProperties.orientation;o===0&&(s==="landscape"||s!=="portrait"&&((n.width||0)>(n.height||0)||n.properties?.otherProperties.spread==="both"))&&(this.shift=!1),e&&n.properties?.page===I.center&&this.spreads[o-1][0].addProperties({addBlank:!0}),s==="portrait"&&n.properties?.page!=="center"&&n.properties?.otherProperties.number>1?e=!0:e=!1}),this.shift||this.index(t,!0)}buildSpreads(t){let e=[];t.items.forEach((i,o)=>{!o&&this.shift?this.spreads.push([i]):i.properties?.page===I.center?(e.length>0&&this.spreads.push(e),this.spreads.push([i]),e=[]):e.length>=2?(this.spreads.push(e),e=[i]):e.push(i)}),e.length>0&&this.spreads.push(e)}currentSpread(t,e){return this.spreads[Math.min(Math.floor(t/e),this.spreads.length-1)]}findByLink(t){return this.spreads.find(e=>e.includes(t))||void 0}};var Mr=8,Nr=5,Ta=300,za=15e3,La=250,Ar=150,Dr=500,zi=class{constructor(t,e,i,o,n,s,a,h=[],l){if(this.pool=new Map,this.blobs=new Map,this.inprogress=new Map,this.delayedShow=new Map,this.delayedTimeout=new Map,this.previousFrames=[],this.injector=null,this.width=0,this.height=0,this.currentSlide=0,this.spread=!0,this.orientationInternal=-1,this.bookScale=null,this.bookTranslate=null,this.bookTransform=null,this.container=t,this.positions=e,this.pub=i,this.injector=n??null,this.contentProtectionConfig=s||{},this.keyboardPeripheralsConfig=a||[],this.resizeWatchSelectors=[...h],this.upgradeInsecureRequests=l??"always",this.spreadPresentation=i.metadata.otherMetadata?.spread||Ti.auto,this.pub.metadata.effectiveReadingProgression!==E.rtl&&this.pub.metadata.effectiveReadingProgression!==E.ltr)throw Error("Unsupported reading progression for fixed-layout EPUB");this.spreader=new xi(this.pub),this.containerHeightCached=t.clientHeight,this.bookElement=document.createElement("div"),this.bookElement.ariaLabel="Book",this.bookElement.tabIndex=-1,this.updateBookStyle(!0),this.spineElement=document.createElement("div"),this.spineElement.ariaLabel="Spine",this.bookElement.appendChild(this.spineElement),this.container.appendChild(this.bookElement),this.updateSpineStyle(!0),this.peripherals=new Pi(this);let d=document.createDocumentFragment();this.pub.readingOrder.items.forEach(c=>{let m=new Fi(this.peripherals,this.pub.metadata.effectiveReadingProgression,c.href,this.contentProtectionConfig,this.keyboardPeripheralsConfig,this.resizeWatchSelectors);this.pool.set(c.href,m),m.width=100/this.length*(c.properties?.otherProperties.orientation===ki.landscape||c.properties?.otherProperties.addBlank?this.perPage:1),m.height=this.height,d.appendChild(m.wrapper)}),this.spineElement.appendChild(d)}set listener(t){this._listener=t}get listener(){return this._listener}get doNotDisturb(){return this.peripherals.pan.touchID>0}resizeHandler(t=!0,e=!0){this.currentSlide+this.perPage>this.length&&(this.currentSlide=this.length<=this.perPage?0:this.length-1),this.containerHeightCached=this.container.clientHeight,this.orientationInternal=-1,this.updateSpineStyle(!0),t&&(this.currentSlide=this.reAlign(),this.slideToCurrent(!e,e)),clearTimeout(this.resizeTimeout),this.resizeTimeout=window.setTimeout(()=>{let i=100/this.length,o=new Map(this.pub.readingOrder.items.map(n=>[n.href,n]));this.pool.forEach((n,s)=>{let a=o.get(s);a&&requestAnimationFrame(()=>{if(n.width=i*(a.properties?.otherProperties.orientation===ki.landscape||a.properties?.otherProperties.addBlank?this.perPage:1),n.height=this.height,!n.loaded)return;let h=this.spreader.findByLink(a);n.update(this.spreadPosition(h,a))})})},La)}updateDimensions(){this.width=this.bookElement.clientWidth,this.height=this.bookElement.clientHeight}get rtl(){return this.pub.metadata.effectiveReadingProgression===E.rtl}get single(){return!this.spread||this.portrait}get perPage(){return this.spread&&!this.portrait?2:1}get threshold(){return 50}get portrait(){return this.spreadPresentation===Ti.none?!0:(this.orientationInternal===-1&&(this.orientationInternal=this.containerHeightCached>this.container.clientWidth?1:0),this.orientationInternal===1)}updateSpineStyle(t,e=!0){let i=0;if(this.updateDimensions(),this.perPage>1&&(i=this.width/2),mt())this.spineElement.attributeStyleMap.set("transition",new CSSUnparsedValue([t?`all ${e?Ar:Dr}ms ease-out`:"all 0ms ease-out"])),this.spineElement.attributeStyleMap.set("margin-right",CSS.px(this.rtl?i:0)),this.spineElement.attributeStyleMap.set("margin-left",CSS.px(this.rtl?0:i)),this.spineElement.attributeStyleMap.set("width",CSS.px(this.width/this.perPage*this.length)),this.transform?this.spineElement.attributeStyleMap.set("transform",new CSSTransformValue([new CSSTranslate(CSS.px(this.transform||0),CSS.px(0),CSS.px(0))])):this.spineElement.attributeStyleMap.delete("transform"),this.spineElement.attributeStyleMap.set("contain",new CSSUnparsedValue(["content"]));else{let o={transition:t?`all ${e?Ar:Dr}ms ease-out`:"all 0ms ease-out",marginRight:this.rtl?`${i}px`:"0",marginLeft:this.rtl?"0":`${i}px`,width:`${this.width/this.perPage*this.length}px`,transform:this.transform?`translate3d(${this.transform}px, 0px, 0px)`:"",contain:"content"};Object.assign(this.spineElement.style,o)}}updateBookStyle(t=!1){if(t){let n={overflow:"hidden",direction:this.pub.metadata.effectiveReadingProgression,cursor:"",height:"100%",width:"100%",position:"relative",outline:"none",transition:this.peripherals?.dragState?"none":"transform .15s ease-in-out",touchAction:"none"};Object.assign(this.bookElement.style,n)}let e=this.peripherals?.scale||1,i=this.peripherals?.pan.translateX||0,o=this.peripherals?.pan.translateY||0;mt()?(this.bookTransform||(this.bookScale=new CSSScale(1,1),this.bookTranslate=new CSSTranslate(CSS.px(0),CSS.px(0),CSS.px(0)),this.bookTransform=new CSSTransformValue([this.bookScale,this.bookTranslate])),this.bookScale.x.value=e,this.bookScale.y.value=e,this.bookTranslate.x.value=i,this.bookTranslate.y.value=o,this.bookElement.attributeStyleMap.set("transform",this.bookTransform)):this.bookElement.style.transform=`scale(${e}) translate3d(${i}px, ${o}px, 0px)`}goTo(t){if(this.slength<=this.perPage)return;t=this.reAlign(t);let e=this.currentSlide;this.currentSlide=Math.min(Math.max(t,0),this.length-1),e!==this.currentSlide&&this.slideToCurrent(!1)}onChange(){this.peripherals.scale=1,this.updateBookStyle()}get offset(){return(this.rtl?1:-1)*this.currentSlide*(this.width/this.perPage)}get length(){if(this.single)return this.slength;let t=this.slength+this.nLandscape;return this.shift&&t%2===0?t+1:t}get slength(){return this.pub.readingOrder.items.length||0}get shift(){return this.spreader.shift}get nLandscape(){return this.spreader.nLandscape}setPerPage(t){t===null?this.spread=!0:t===1?this.spread=!1:this.spread=!0,requestAnimationFrame(()=>this.resizeHandler(!0))}slideToCurrent(t,e=!0){this.updateDimensions(),t?requestAnimationFrame(()=>{requestAnimationFrame(()=>{this.transform=this.offset,this.updateSpineStyle(!0,e),this.deselect()})}):(this.transform=this.offset,this.updateSpineStyle(!1),this.deselect())}bounce(t=!1){requestAnimationFrame(()=>{this.transform=this.offset+50*(t?1:-1),this.updateSpineStyle(!0,!0),setTimeout(()=>{this.transform=this.offset,this.updateSpineStyle(!0,!0)},100)})}next(t=1){if(this.slength<=this.perPage)return!1;let e=this.currentSlide;return this.currentSlide=Math.min(this.currentSlide+t,this.length-1),this.perPage>1&&this.currentSlide%2&&this.currentSlide--,this.currentSlide===e&&(this.currentSlide+1,this.length),e!==this.currentSlide?(this.slideToCurrent(!0),this.onChange(),!0):(this.bounce(this.rtl),!1)}prev(t=1){if(this.slength<=this.perPage)return!1;let e=this.currentSlide;return this.currentSlide=Math.max(this.currentSlide-t,0),this.perPage>1&&this.currentSlide%2&&this.currentSlide++,e!==this.currentSlide?(this.slideToCurrent(!0),this.onChange(),!0):(this.bounce(!this.rtl),!1)}get ownerWindow(){return this.container.ownerDocument.defaultView||window}async destroy(){let t=this.inprogress.values(),e=t.next(),i=[];for(;e.value;)i.push(e.value),e=t.next();i.length>0&&await Promise.allSettled(i),this.inprogress.clear();let o=this.pool.values(),n=o.next();for(;n.value;)await n.value.destroy(),n=o.next();this.pool.clear(),this.blobs.forEach(s=>s.reset()),this.blobs.clear(),this.injector?.dispose(),this.container.childNodes.forEach(s=>{(s.nodeType===Node.ELEMENT_NODE||s.nodeType===Node.TEXT_NODE)&&s.remove()})}makeSpread(t){return this.perPage<2?[this.pub.readingOrder.items[t]]:this.spreader.currentSpread(t,this.perPage)}reAlign(t=this.currentSlide){return t%2&&!this.single&&t++,t}spreadPosition(t,e){return this.perPage<2?I.center:t.length<2?I.center:e.href===t[0].href?this.rtl?I.right:I.left:this.rtl?I.left:I.right}async waitForItem(t){if(this.inprogress.has(t)&&await this.inprogress.get(t),this.delayedShow.has(t)){let e=this.delayedTimeout.get(t);e>0?clearTimeout(e):await this.delayedShow.get(t),this.delayedTimeout.set(t,0),this.delayedShow.delete(t)}}async cancelShowing(t){if(this.delayedShow.has(t)){let e=this.delayedTimeout.get(t);e>0&&clearTimeout(e),this.delayedShow.delete(t)}}async update(t,e,i,o=!1){let n=this.pub.readingOrder.items.findIndex(h=>h.href===e.href);if(n<0)throw Error("Href not found in reading order");this.currentSlide!==n&&(this.currentSlide=this.reAlign(n),this.slideToCurrent(!0));let s=this.makeSpread(this.currentSlide);this.perPage>1&&n++;for(let h of s)await this.waitForItem(h.href);let a=new Promise(async(h,l)=>{let d=[],c=[];this.positions.forEach((u,f)=>{(f>n+Mr||f<n-Mr)&&(d.includes(u.href)||d.push(u.href)),f<n+Nr&&f>n-Nr&&(c.includes(u.href)||c.push(u.href))}),d.forEach(async u=>{c.includes(u)||this.pool.has(u)&&(this.cancelShowing(u),await this.pool.get(u)?.unload(),this.blobs.get(u)?.reset())}),this.currentBaseURL!==void 0&&t.baseURL!==this.currentBaseURL&&(this.blobs.forEach(u=>u.reset()),this.blobs.clear()),this.currentBaseURL=t.baseURL;let m=async u=>{let f=t.readingOrder.findIndexWithHref(u),S=t.readingOrder.items[f];S&&(this.blobs.has(u)||this.blobs.set(u,new ue(t,this.currentBaseURL||"",S,{injector:this.injector,upgradeInsecureRequests:this.upgradeInsecureRequests})),this.delayedShow.has(u)||this.delayedShow.set(u,new Promise((b,C)=>{let v=!1,w=window.setTimeout(async()=>{this.delayedTimeout.set(u,0);let B=this.makeSpread(this.reAlign(f)),it=this.spreadPosition(B,S),lt=this.pool.get(u);await lt.load(i,await this.blobs.get(u).build(!0)),this.peripherals.isScaled||await lt.show(it),this.delayedShow.delete(u),v=!0,b()},Ta);setTimeout(()=>{!v&&this.delayedShow.has(u)&&C(`Offscreen load timeout: ${u}`)},za),this.delayedTimeout.set(u,w)})))};try{await Promise.all(c.map(u=>m(u)))}catch(u){l(u)}let p=[];for(let u of s){let f=this.pool.get(u.href),S=this.blobs.get(u.href);S&&(this.cancelShowing(u.href),await f.load(i,await S.build(!0)),await f.show(this.spreadPosition(s,u)),this.previousFrames.push(f),await f.activate(),p.push(f))}for(;this.previousFrames.length>0;){let u=this.previousFrames.shift();u&&!p.includes(u)&&await u.unfocus()}this.previousFrames=p;let g=this.container.ownerDocument.activeElement;g&&g.tagName==="IFRAME"&&!p.some(u=>u.iframe===g)&&p[0]?.iframe.focus({preventScroll:!0}),h()});for(let h of s)this.inprogress.set(h.href,a);try{await a}finally{for(let h of s)this.inprogress.delete(h.href)}}addResizeTarget(t){this.resizeWatchSelectors.includes(t)||this.resizeWatchSelectors.push(t),this.pool.forEach(e=>e.addResizeTarget(t))}removeResizeTarget(t){this.resizeWatchSelectors=this.resizeWatchSelectors.filter(e=>e!==t),this.pool.forEach(e=>e.removeResizeTarget(t))}get currentFrames(){if(this.perPage<2){let t=this.pub.readingOrder.items[this.currentSlide];return[this.pool.get(t.href)]}return this.spreader.currentSpread(this.currentSlide,this.perPage).map(t=>this.pool.get(t.href))}get currentBounds(){let t={x:0,y:0,width:0,height:0,top:0,right:0,bottom:0,left:0,toJSON(){return this}};return this.currentFrames.forEach(e=>{if(!e)return;let i=e.realSize;t.x=Math.min(t.x,i.x),t.y=Math.min(t.y,i.y),t.width+=i.width,t.height=Math.max(t.height,i.height),t.top=Math.min(t.top,i.top),t.right=Math.min(t.right,i.right),t.bottom=Math.min(t.bottom,i.bottom),t.left=Math.min(t.left,i.left)}),t}get viewport(){let t={readingOrder:[],progressions:new Map,positions:null};return(this.perPage<2?[this.pub.readingOrder.items[this.currentSlide]]:this.spreader.currentSpread(this.currentSlide,this.perPage)).forEach(e=>{t.readingOrder.push(e.href),t.progressions.set(e.href,{start:0,end:1})}),t.positions=this.getCurrentNumbers(),t}getCurrentNumbers(){if(this.perPage<2)return[this.pub.readingOrder.items[this.currentSlide].properties?.otherProperties.number];let t=this.spreader.currentSpread(this.currentSlide,this.perPage);return t.length>1?[t[0].properties?.otherProperties.number,t[t.length-1].properties?.otherProperties.number]:[t[0].properties?.otherProperties.number]}deselect(){this.currentFrames?.forEach(t=>t?.deselect())}};var Yt=class r{constructor(t={}){this.backgroundColor=K(t.backgroundColor),this.blendFilter=L(t.blendFilter),this.constraint=P(t.constraint),this.columnCount=P(t.columnCount),this.darkenFilter=kt(t.darkenFilter),this.deprecatedFontSize=L(t.deprecatedFontSize),this.fontFamily=K(t.fontFamily),this.fontSize=Tt(t.fontSize,Vt.range),this.fontSizeNormalize=L(t.fontSizeNormalize),this.fontOpticalSizing=L(t.fontOpticalSizing),this.fontWeight=Tt(t.fontWeight,Xt.range),this.fontWidth=Tt(t.fontWidth,$t.range),this.hyphens=L(t.hyphens),this.invertFilter=kt(t.invertFilter),this.invertGaijiFilter=kt(t.invertGaijiFilter),this.iOSPatch=L(t.iOSPatch),this.iPadOSPatch=L(t.iPadOSPatch),this.letterSpacing=P(t.letterSpacing),this.ligatures=L(t.ligatures),this.lineHeight=P(t.lineHeight),this.linkColor=K(t.linkColor),this.noRuby=L(t.noRuby),this.pageGutter=P(t.pageGutter),this.paragraphIndent=P(t.paragraphIndent),this.paragraphSpacing=P(t.paragraphSpacing),this.scroll=L(t.scroll),this.scrollPaddingTop=P(t.scrollPaddingTop),this.scrollPaddingBottom=P(t.scrollPaddingBottom),this.scrollPaddingLeft=P(t.scrollPaddingLeft),this.scrollPaddingRight=P(t.scrollPaddingRight),this.selectionBackgroundColor=K(t.selectionBackgroundColor),this.selectionTextColor=K(t.selectionTextColor),this.textAlign=hi(t.textAlign,xt),this.textColor=K(t.textColor),this.textNormalization=L(t.textNormalization),this.visitedColor=K(t.visitedColor),this.wordSpacing=P(t.wordSpacing),this.optimalLineLength=P(t.optimalLineLength),this.maximalLineLength=P(t.maximalLineLength),this.minimalLineLength=P(t.minimalLineLength)}static serialize(t){let{...e}=t;return JSON.stringify(e)}static deserialize(t){try{let e=JSON.parse(t);return new r(e)}catch(e){return console.error("Failed to deserialize preferences:",e),null}}merging(t){let e={...this};for(let i of Object.keys(t))t[i]!==void 0&&(i!=="maximalLineLength"||t[i]===null||t[i]>=(t.optimalLineLength??e.optimalLineLength??65))&&(i!=="minimalLineLength"||t[i]===null||t[i]<=(t.optimalLineLength??e.optimalLineLength??65))&&(e[i]=t[i]);return new r(e)}};var Li=class{constructor(t){this.backgroundColor=K(t.backgroundColor)||null,this.blendFilter=L(t.blendFilter)??!1,this.constraint=P(t.constraint)||0,this.columnCount=P(t.columnCount)||null,this.darkenFilter=kt(t.darkenFilter)??!1,this.deprecatedFontSize=L(t.deprecatedFontSize),(this.deprecatedFontSize===!1||this.deprecatedFontSize===null)&&(this.deprecatedFontSize=!CSS.supports("zoom","1")),this.fontFamily=K(t.fontFamily)||null,this.fontSize=Tt(t.fontSize,Vt.range)||1,this.fontSizeNormalize=L(t.fontSizeNormalize)??!1,this.fontOpticalSizing=L(t.fontOpticalSizing)??null,this.fontWeight=Tt(t.fontWeight,Xt.range)||null,this.fontWidth=Tt(t.fontWidth,$t.range)||null,this.hyphens=L(t.hyphens)??null,this.invertFilter=kt(t.invertFilter)??!1,this.invertGaijiFilter=kt(t.invertGaijiFilter)??!1,this.iOSPatch=t.iOSPatch===!1?!1:(st.OS.iOS||st.OS.iPadOS)&&st.iOSRequest==="mobile",this.iPadOSPatch=t.iPadOSPatch===!1?!1:st.OS.iPadOS&&st.iOSRequest==="desktop",this.letterSpacing=P(t.letterSpacing)||null,this.ligatures=L(t.ligatures)??null,this.lineHeight=P(t.lineHeight)||null,this.linkColor=K(t.linkColor)||null,this.noRuby=L(t.noRuby)??!1,this.pageGutter=ci(P(t.pageGutter),20),this.paragraphIndent=P(t.paragraphIndent)??null,this.paragraphSpacing=P(t.paragraphSpacing)??null,this.scroll=L(t.scroll)??!1,this.scrollPaddingTop=P(t.scrollPaddingTop)??null,this.scrollPaddingBottom=P(t.scrollPaddingBottom)??null,this.scrollPaddingLeft=P(t.scrollPaddingLeft)??null,this.scrollPaddingRight=P(t.scrollPaddingRight)??null,this.selectionBackgroundColor=K(t.selectionBackgroundColor)||null,this.selectionTextColor=K(t.selectionTextColor)||null,this.textAlign=hi(t.textAlign,xt)||null,this.textColor=K(t.textColor)||null,this.textNormalization=L(t.textNormalization)??!1,this.visitedColor=K(t.visitedColor)||null,this.wordSpacing=P(t.wordSpacing)||null,this.optimalLineLength=P(t.optimalLineLength)||65,this.maximalLineLength=ci(yr(t.maximalLineLength,this.optimalLineLength),80),this.minimalLineLength=ci(Sr(t.minimalLineLength,this.optimalLineLength),40),this.experiments=_r(t.experiments)||null}};var Oa="#FFFFFF",Ma="#121212",Na="#0000EE",Aa="#551A8B",Da="#b4d8fe",Ua="inherit",qt={backgroundColor:Oa,textColor:Ma,linkColor:Na,visitedColor:Aa,selectionBackgroundColor:Da,selectionTextColor:Ua};var Te=class{constructor(t,e,i){this.preferences=new Yt({...t}),this.settings=e,this.metadata=i,this.layout=this.metadata?.effectiveLayout||_.reflowable}clear(){this.preferences=new Yt({backgroundColor:null,blendFilter:null,columnCount:null,constraint:null,darkenFilter:null,deprecatedFontSize:null,fontFamily:null,fontSize:null,fontSizeNormalize:null,fontOpticalSizing:null,fontWeight:null,fontWidth:null,hyphens:null,invertFilter:null,invertGaijiFilter:null,iOSPatch:null,iPadOSPatch:null,letterSpacing:null,ligatures:null,lineHeight:null,linkColor:null,maximalLineLength:null,minimalLineLength:null,noRuby:null,optimalLineLength:null,pageGutter:null,paragraphIndent:null,paragraphSpacing:null,scroll:null,scrollPaddingTop:null,scrollPaddingBottom:null,scrollPaddingLeft:null,scrollPaddingRight:null,selectionBackgroundColor:null,selectionTextColor:null,textAlign:null,textColor:null,textNormalization:null,visitedColor:null,wordSpacing:null})}updatePreference(t,e){this.preferences[t]=e}get backgroundColor(){return new N({initialValue:this.preferences.backgroundColor,effectiveValue:this.settings.backgroundColor||qt.backgroundColor,isEffective:this.preferences.backgroundColor!==null,onChange:t=>{this.updatePreference("backgroundColor",t??null)}})}get blendFilter(){return new tt({initialValue:this.preferences.blendFilter,effectiveValue:this.settings.blendFilter||!1,isEffective:this.preferences.blendFilter!==null,onChange:t=>{this.updatePreference("blendFilter",t??null)}})}get columnCount(){return new N({initialValue:this.preferences.columnCount,effectiveValue:this.settings.columnCount||null,isEffective:this.layout!==_.fixed&&!this.settings.scroll,onChange:t=>{this.updatePreference("columnCount",t??null)}})}get constraint(){return new N({initialValue:this.preferences.constraint,effectiveValue:this.preferences.constraint||0,isEffective:!0,onChange:t=>{this.updatePreference("constraint",t??null)}})}get darkenFilter(){return new W({initialValue:typeof this.preferences.darkenFilter=="boolean"?100:this.preferences.darkenFilter,effectiveValue:typeof this.settings.darkenFilter=="boolean"?100:this.settings.darkenFilter||0,isEffective:this.settings.darkenFilter!==null,onChange:t=>{this.updatePreference("darkenFilter",t??null)},supportedRange:Gt.range,step:Gt.step})}get deprecatedFontSize(){return new tt({initialValue:this.preferences.deprecatedFontSize,effectiveValue:CSS.supports("zoom","1")?this.settings.deprecatedFontSize||!1:!0,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("deprecatedFontSize",t??null)}})}get fontFamily(){return new N({initialValue:this.preferences.fontFamily,effectiveValue:this.settings.fontFamily||null,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("fontFamily",t??null)}})}get fontSize(){return new W({initialValue:this.preferences.fontSize,effectiveValue:this.settings.fontSize||1,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("fontSize",t??null)},supportedRange:Vt.range,step:Vt.step})}get fontSizeNormalize(){return new tt({initialValue:this.preferences.fontSizeNormalize,effectiveValue:this.settings.fontSizeNormalize||!1,isEffective:this.layout!==_.fixed&&this.preferences.fontSizeNormalize!==null,onChange:t=>{this.updatePreference("fontSizeNormalize",t??null)}})}get fontOpticalSizing(){return new tt({initialValue:this.preferences.fontOpticalSizing,effectiveValue:this.settings.fontOpticalSizing||!0,isEffective:this.layout!==_.fixed&&this.preferences.fontOpticalSizing!==null,onChange:t=>{this.updatePreference("fontOpticalSizing",t??null)}})}get fontWeight(){return new W({initialValue:this.preferences.fontWeight,effectiveValue:this.settings.fontWeight||400,isEffective:this.layout!==_.fixed&&this.preferences.fontWeight!==null,onChange:t=>{this.updatePreference("fontWeight",t??null)},supportedRange:Xt.range,step:Xt.step})}get fontWidth(){return new W({initialValue:this.preferences.fontWidth,effectiveValue:this.settings.fontWidth||100,isEffective:this.layout!==_.fixed&&this.preferences.fontWidth!==null,onChange:t=>{this.updatePreference("fontWidth",t??null)},supportedRange:$t.range,step:$t.step})}get hyphens(){return new tt({initialValue:this.preferences.hyphens,effectiveValue:this.settings.hyphens||!1,isEffective:this.layout!==_.fixed&&this.metadata?.effectiveReadingProgression===E.ltr&&this.preferences.hyphens!==null,onChange:t=>{this.updatePreference("hyphens",t??null)}})}get invertFilter(){return new W({initialValue:typeof this.preferences.invertFilter=="boolean"?100:this.preferences.invertFilter,effectiveValue:typeof this.settings.invertFilter=="boolean"?100:this.settings.invertFilter||0,isEffective:this.settings.invertFilter!==null,onChange:t=>{this.updatePreference("invertFilter",t??null)},supportedRange:Gt.range,step:Gt.step})}get invertGaijiFilter(){return new W({initialValue:typeof this.preferences.invertGaijiFilter=="boolean"?100:this.preferences.invertGaijiFilter,effectiveValue:typeof this.settings.invertGaijiFilter=="boolean"?100:this.settings.invertGaijiFilter||0,isEffective:this.preferences.invertGaijiFilter!==null,onChange:t=>{this.updatePreference("invertGaijiFilter",t??null)},supportedRange:Gt.range,step:Gt.step})}get iOSPatch(){return new tt({initialValue:this.preferences.iOSPatch,effectiveValue:this.settings.iOSPatch||!1,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("iOSPatch",t??null)}})}get iPadOSPatch(){return new tt({initialValue:this.preferences.iPadOSPatch,effectiveValue:this.settings.iPadOSPatch||!1,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("iPadOSPatch",t??null)}})}get letterSpacing(){return new W({initialValue:this.preferences.letterSpacing,effectiveValue:this.settings.letterSpacing||0,isEffective:this.layout!==_.fixed&&this.preferences.letterSpacing!==null,onChange:t=>{this.updatePreference("letterSpacing",t??null)},supportedRange:Lo.range,step:Lo.step})}get ligatures(){return new tt({initialValue:this.preferences.ligatures,effectiveValue:this.settings.ligatures||!0,isEffective:(()=>{if(this.preferences.ligatures===null||this.layout===_.fixed)return!1;let t=this.metadata?.languages?.[0]?.toLowerCase();return!(t&&["zh","ja","ko","mn-mong"].some(e=>t.startsWith(e)))})(),onChange:t=>{this.updatePreference("ligatures",t??null)}})}get lineHeight(){return new W({initialValue:this.preferences.lineHeight,effectiveValue:this.settings.lineHeight,isEffective:this.layout!==_.fixed&&this.preferences.lineHeight!==null,onChange:t=>{this.updatePreference("lineHeight",t??null)},supportedRange:Oo.range,step:Oo.step})}get linkColor(){return new N({initialValue:this.preferences.linkColor,effectiveValue:this.settings.linkColor||qt.linkColor,isEffective:this.layout!==_.fixed&&this.preferences.linkColor!==null,onChange:t=>{this.updatePreference("linkColor",t??null)}})}get maximalLineLength(){return new W({initialValue:this.preferences.maximalLineLength,effectiveValue:this.settings.maximalLineLength,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("maximalLineLength",t)},supportedRange:Kt.range,step:Kt.step})}get minimalLineLength(){return new W({initialValue:this.preferences.minimalLineLength,effectiveValue:this.settings.minimalLineLength,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("minimalLineLength",t)},supportedRange:Kt.range,step:Kt.step})}get noRuby(){return new tt({initialValue:this.preferences.noRuby,effectiveValue:this.settings.noRuby||!1,isEffective:this.layout!==_.fixed&&this.metadata?.languages?.includes("ja")||!1,onChange:t=>{this.updatePreference("noRuby",t??null)}})}get optimalLineLength(){return new W({initialValue:this.preferences.optimalLineLength,effectiveValue:this.settings.optimalLineLength,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("optimalLineLength",t)},supportedRange:Kt.range,step:Kt.step})}get pageGutter(){return new N({initialValue:this.preferences.pageGutter,effectiveValue:this.settings.pageGutter,isEffective:this.layout!==_.fixed&&!this.settings.scroll,onChange:t=>{this.updatePreference("pageGutter",t??null)}})}get paragraphIndent(){return new W({initialValue:this.preferences.paragraphIndent,effectiveValue:this.settings.paragraphIndent||0,isEffective:this.layout!==_.fixed&&this.preferences.paragraphIndent!==null,onChange:t=>{this.updatePreference("paragraphIndent",t??null)},supportedRange:Mo.range,step:Mo.step})}get paragraphSpacing(){return new W({initialValue:this.preferences.paragraphSpacing,effectiveValue:this.settings.paragraphSpacing||0,isEffective:this.layout!==_.fixed&&this.preferences.paragraphSpacing!==null,onChange:t=>{this.updatePreference("paragraphSpacing",t??null)},supportedRange:No.range,step:No.step})}get scroll(){return new tt({initialValue:this.preferences.scroll,effectiveValue:this.settings.scroll||!1,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("scroll",t??null)}})}get scrollPaddingTop(){return new N({initialValue:this.preferences.scrollPaddingTop,effectiveValue:this.settings.scrollPaddingTop||0,isEffective:this.layout!==_.fixed&&!!this.settings.scroll&&this.preferences.scrollPaddingTop!==null,onChange:t=>{this.updatePreference("scrollPaddingTop",t??null)}})}get scrollPaddingBottom(){return new N({initialValue:this.preferences.scrollPaddingBottom,effectiveValue:this.settings.scrollPaddingBottom||0,isEffective:this.layout!==_.fixed&&!!this.settings.scroll&&this.preferences.scrollPaddingBottom!==null,onChange:t=>{this.updatePreference("scrollPaddingBottom",t??null)}})}get scrollPaddingLeft(){return new N({initialValue:this.preferences.scrollPaddingLeft,effectiveValue:this.settings.scrollPaddingLeft||0,isEffective:this.layout!==_.fixed&&!!this.settings.scroll&&this.preferences.scrollPaddingLeft!==null,onChange:t=>{this.updatePreference("scrollPaddingLeft",t??null)}})}get scrollPaddingRight(){return new N({initialValue:this.preferences.scrollPaddingRight,effectiveValue:this.settings.scrollPaddingRight||0,isEffective:this.layout!==_.fixed&&!!this.settings.scroll&&this.preferences.scrollPaddingRight!==null,onChange:t=>{this.updatePreference("scrollPaddingRight",t??null)}})}get selectionBackgroundColor(){return new N({initialValue:this.preferences.selectionBackgroundColor,effectiveValue:this.settings.selectionBackgroundColor||qt.selectionBackgroundColor,isEffective:this.layout!==_.fixed&&this.preferences.selectionBackgroundColor!==null,onChange:t=>{this.updatePreference("selectionBackgroundColor",t??null)}})}get selectionTextColor(){return new N({initialValue:this.preferences.selectionTextColor,effectiveValue:this.settings.selectionTextColor||qt.selectionTextColor,isEffective:this.layout!==_.fixed&&this.preferences.selectionTextColor!==null,onChange:t=>{this.updatePreference("selectionTextColor",t??null)}})}get textAlign(){return new di({initialValue:this.preferences.textAlign,effectiveValue:this.settings.textAlign||xt.start,isEffective:this.layout!==_.fixed&&this.preferences.textAlign!==null,onChange:t=>{this.updatePreference("textAlign",t??null)},supportedValues:Object.values(xt)})}get textColor(){return new N({initialValue:this.preferences.textColor,effectiveValue:this.settings.textColor||qt.textColor,isEffective:this.layout!==_.fixed&&this.preferences.textColor!==null,onChange:t=>{this.updatePreference("textColor",t??null)}})}get textNormalization(){return new tt({initialValue:this.preferences.textNormalization,effectiveValue:this.settings.textNormalization||!1,isEffective:this.layout!==_.fixed,onChange:t=>{this.updatePreference("textNormalization",t??null)}})}get visitedColor(){return new N({initialValue:this.preferences.visitedColor,effectiveValue:this.settings.visitedColor||qt.visitedColor,isEffective:this.layout!==_.fixed&&this.preferences.visitedColor!==null,onChange:t=>{this.updatePreference("visitedColor",t??null)}})}get wordSpacing(){return new W({initialValue:this.preferences.wordSpacing,effectiveValue:this.settings.wordSpacing||0,isEffective:this.layout!==_.fixed&&this.preferences.wordSpacing!==null,onChange:t=>{this.updatePreference("wordSpacing",t??null)},supportedRange:Ao.range,step:Ao.step})}};function Jt(r){let t=getComputedStyle(r),e=parseFloat(t.paddingLeft||"0"),i=parseFloat(t.paddingRight||"0");return r.clientWidth-e-i}function Ur(r){let t=getComputedStyle(r),e=parseFloat(t.paddingTop||"0"),i=parseFloat(t.paddingBottom||"0");return r.clientHeight-e-i}var pe=class extends xe{constructor(t){super(),this.a11yNormalize=t.a11yNormalize??null,this.backgroundColor=t.backgroundColor??null,this.blendFilter=t.blendFilter??null,this.bodyHyphens=t.bodyHyphens??null,this.colCount=t.colCount??null,this.darkenFilter=t.darkenFilter??null,this.deprecatedFontSize=t.deprecatedFontSize??null,this.fontFamily=t.fontFamily??null,this.fontOpticalSizing=t.fontOpticalSizing??null,this.fontSize=t.fontSize??null,this.fontSizeNormalize=t.fontSizeNormalize??null,this.fontWeight=t.fontWeight??null,this.fontWidth=t.fontWidth??null,this.invertFilter=t.invertFilter??null,this.invertGaijiFilter=t.invertGaijiFilter??null,this.iOSPatch=t.iOSPatch??null,this.iPadOSPatch=t.iPadOSPatch??null,this.letterSpacing=t.letterSpacing??null,this.ligatures=t.ligatures??null,this.lineHeight=t.lineHeight??null,this.lineLength=t.lineLength??null,this.linkColor=t.linkColor??null,this.noRuby=t.noRuby??null,this.paraIndent=t.paraIndent??null,this.paraSpacing=t.paraSpacing??null,this.selectionBackgroundColor=t.selectionBackgroundColor??null,this.selectionTextColor=t.selectionTextColor??null,this.textAlign=t.textAlign??null,this.textColor=t.textColor??null,this.view=t.view??null,this.visitedColor=t.visitedColor??null,this.wordSpacing=t.wordSpacing??null}toCSSProperties(){let t={};return this.a11yNormalize&&(t["--USER__a11yNormalize"]=this.toFlag("a11y")),this.backgroundColor&&(t["--USER__backgroundColor"]=this.backgroundColor),this.blendFilter&&(t["--USER__blendFilter"]=this.toFlag("blend")),this.bodyHyphens&&(t["--USER__bodyHyphens"]=this.bodyHyphens),this.colCount&&(t["--USER__colCount"]=this.toUnitless(this.colCount)),this.darkenFilter===!0?t["--USER__darkenFilter"]=this.toFlag("darken"):typeof this.darkenFilter=="number"&&(t["--USER__darkenFilter"]=this.toPercentage(this.darkenFilter)),this.deprecatedFontSize&&(t["--USER__fontSizeImplementation"]=this.toFlag("deprecatedFontSize")),this.fontFamily&&(t["--USER__fontFamily"]=this.fontFamily),this.fontOpticalSizing!=null&&(t["--USER__fontOpticalSizing"]=this.fontOpticalSizing),this.fontSize!=null&&(t["--USER__fontSize"]=this.toPercentage(this.fontSize,!0)),this.fontSizeNormalize&&(t["--USER__fontSizeNormalize"]=this.toFlag("normalize")),this.fontWeight!=null&&(t["--USER__fontWeight"]=this.toUnitless(this.fontWeight)),this.fontWidth!=null&&(t["--USER__fontWidth"]=typeof this.fontWidth=="string"?this.fontWidth:this.toUnitless(this.fontWidth)),this.invertFilter===!0?t["--USER__invertFilter"]=this.toFlag("invert"):typeof this.invertFilter=="number"&&(t["--USER__invertFilter"]=this.toPercentage(this.invertFilter)),this.invertGaijiFilter===!0?t["--USER__invertGaiji"]=this.toFlag("invertGaiji"):typeof this.invertGaijiFilter=="number"&&(t["--USER__invertGaiji"]=this.toPercentage(this.invertGaijiFilter)),this.iOSPatch&&(t["--USER__iOSPatch"]=this.toFlag("iOSPatch")),this.iPadOSPatch&&(t["--USER__iPadOSPatch"]=this.toFlag("iPadOSPatch")),this.letterSpacing!=null&&(t["--USER__letterSpacing"]=this.toRem(this.letterSpacing)),this.ligatures&&(t["--USER__ligatures"]=this.ligatures),this.lineHeight!=null&&(t["--USER__lineHeight"]=this.toUnitless(this.lineHeight)),this.lineLength!=null&&(t["--USER__lineLength"]=this.toPx(this.lineLength)),this.linkColor&&(t["--USER__linkColor"]=this.linkColor),this.noRuby&&(t["--USER__noRuby"]=this.toFlag("noRuby")),this.paraIndent!=null&&(t["--USER__paraIndent"]=this.toRem(this.paraIndent)),this.paraSpacing!=null&&(t["--USER__paraSpacing"]=this.toRem(this.paraSpacing)),this.selectionBackgroundColor&&(t["--USER__selectionBackgroundColor"]=this.selectionBackgroundColor),this.selectionTextColor&&(t["--USER__selectionTextColor"]=this.selectionTextColor),this.textAlign&&(t["--USER__textAlign"]=this.textAlign),this.textColor&&(t["--USER__textColor"]=this.textColor),this.view&&(t["--USER__view"]=this.toFlag(this.view)),this.visitedColor&&(t["--USER__visitedColor"]=this.visitedColor),this.wordSpacing!=null&&(t["--USER__wordSpacing"]=this.toRem(this.wordSpacing)),t}},Oi=class extends xe{constructor(t){super(),this.backgroundColor=t.backgroundColor??null,this.baseFontFamily=t.baseFontFamily??null,this.baseFontSize=t.baseFontSize??null,this.baseLineHeight=t.baseLineHeight??null,this.boxSizingMedia=t.boxSizingMedia??null,this.boxSizingTable=t.boxSizingTable??null,this.colWidth=t.colWidth??null,this.colCount=t.colCount??null,this.colGap=t.colGap??null,this.codeFontFamily=t.codeFontFamily??null,this.compFontFamily=t.compFontFamily??null,this.defaultLineLength=t.defaultLineLength??null,this.flowSpacing=t.flowSpacing??null,this.humanistTf=t.humanistTf??null,this.linkColor=t.linkColor??null,this.maxMediaWidth=t.maxMediaWidth??null,this.maxMediaHeight=t.maxMediaHeight??null,this.modernTf=t.modernTf??null,this.monospaceTf=t.monospaceTf??null,this.noOverflow=t.noOverflow??null,this.noVerticalPagination=t.noVerticalPagination??null,this.oldStyleTf=t.oldStyleTf??null,this.pageGutter=t.pageGutter??null,this.paraIndent=t.paraIndent??null,this.paraSpacing=t.paraSpacing??null,this.primaryColor=t.primaryColor??null,this.scrollPaddingBottom=t.scrollPaddingBottom??null,this.scrollPaddingLeft=t.scrollPaddingLeft??null,this.scrollPaddingRight=t.scrollPaddingRight??null,this.scrollPaddingTop=t.scrollPaddingTop??null,this.sansSerifJa=t.sansSerifJa??null,this.sansSerifJaV=t.sansSerifJaV??null,this.sansTf=t.sansTf??null,this.secondaryColor=t.secondaryColor??null,this.selectionBackgroundColor=t.selectionBackgroundColor??null,this.selectionTextColor=t.selectionTextColor??null,this.serifJa=t.serifJa??null,this.serifJaV=t.serifJaV??null,this.textColor=t.textColor??null,this.typeScale=t.typeScale??null,this.visitedColor=t.visitedColor??null,this.experiments=t.experiments??null}toCSSProperties(){let t={};return this.backgroundColor&&(t["--RS__backgroundColor"]=this.backgroundColor),this.baseFontFamily&&(t["--RS__baseFontFamily"]=this.baseFontFamily),this.baseFontSize!=null&&(t["--RS__baseFontSize"]=this.toRem(this.baseFontSize)),this.baseLineHeight!=null&&(t["--RS__baseLineHeight"]=this.toUnitless(this.baseLineHeight)),this.boxSizingMedia&&(t["--RS__boxSizingMedia"]=this.boxSizingMedia),this.boxSizingTable&&(t["--RS__boxSizingTable"]=this.boxSizingTable),this.colWidth!=null&&(t["--RS__colWidth"]=this.colWidth),this.colCount!=null&&(t["--RS__colCount"]=this.toUnitless(this.colCount)),this.colGap!=null&&(t["--RS__colGap"]=this.toPx(this.colGap)),this.codeFontFamily&&(t["--RS__codeFontFamily"]=this.codeFontFamily),this.compFontFamily&&(t["--RS__compFontFamily"]=this.compFontFamily),this.defaultLineLength!=null&&(t["--RS__defaultLineLength"]=this.toPx(this.defaultLineLength)),this.flowSpacing!=null&&(t["--RS__flowSpacing"]=this.toRem(this.flowSpacing)),this.humanistTf&&(t["--RS__humanistTf"]=this.humanistTf),this.linkColor&&(t["--RS__linkColor"]=this.linkColor),this.maxMediaWidth&&(t["--RS__maxMediaWidth"]=this.toVw(this.maxMediaWidth)),this.maxMediaHeight&&(t["--RS__maxMediaHeight"]=this.toVh(this.maxMediaHeight)),this.modernTf&&(t["--RS__modernTf"]=this.modernTf),this.monospaceTf&&(t["--RS__monospaceTf"]=this.monospaceTf),this.noOverflow&&(t["--RS__disableOverflow"]=this.toFlag("noOverflow")),this.noVerticalPagination&&(t["--RS__disablePagination"]=this.toFlag("noVerticalPagination")),this.oldStyleTf&&(t["--RS__oldStyleTf"]=this.oldStyleTf),this.pageGutter!=null&&(t["--RS__pageGutter"]=this.toPx(this.pageGutter)),this.paraIndent!=null&&(t["--RS__paraIndent"]=this.toRem(this.paraIndent)),this.paraSpacing!=null&&(t["--RS__paraSpacing"]=this.toRem(this.paraSpacing)),this.primaryColor&&(t["--RS__primaryColor"]=this.primaryColor),this.sansSerifJa&&(t["--RS__sans-serif-ja"]=this.sansSerifJa),this.sansSerifJaV&&(t["--RS__sans-serif-ja-v"]=this.sansSerifJaV),this.sansTf&&(t["--RS__sansTf"]=this.sansTf),this.scrollPaddingBottom!=null&&(t["--RS__scrollPaddingBottom"]=this.toPx(this.scrollPaddingBottom)),this.scrollPaddingLeft!=null&&(t["--RS__scrollPaddingLeft"]=this.toPx(this.scrollPaddingLeft)),this.scrollPaddingRight!=null&&(t["--RS__scrollPaddingRight"]=this.toPx(this.scrollPaddingRight)),this.scrollPaddingTop!=null&&(t["--RS__scrollPaddingTop"]=this.toPx(this.scrollPaddingTop)),this.secondaryColor&&(t["--RS__secondaryColor"]=this.secondaryColor),this.selectionBackgroundColor&&(t["--RS__selectionBackgroundColor"]=this.selectionBackgroundColor),this.selectionTextColor&&(t["--RS__selectionTextColor"]=this.selectionTextColor),this.serifJa&&(t["--RS__serif-ja"]=this.serifJa),this.serifJaV&&(t["--RS__serif-ja-v"]=this.serifJaV),this.textColor&&(t["--RS__textColor"]=this.textColor),this.typeScale&&(t["--RS__typeScale"]=this.toUnitless(this.typeScale)),this.visitedColor&&(t["--RS__visitedColor"]=this.visitedColor),this.experiments&&this.experiments.forEach(e=>{t["--RS__"+e]=li[e].value}),t}};var Mi=class{constructor(t){this.rsProperties=t.rsProperties,this.userProperties=t.userProperties,this.lineLengths=t.lineLengths,this.container=t.container,this.containerParent=t.container.parentElement||document.documentElement,this.constraint=t.constraint,this.isCJKVertical=t.isCJKVertical??!1,this.cachedColCount=t.userProperties.colCount,this.effectiveContainerWidth=Jt(this.containerParent)}update(t){this.cachedColCount=t.columnCount,t.constraint!==this.constraint&&(this.constraint=t.constraint),t.pageGutter!==this.rsProperties.pageGutter&&(this.rsProperties.pageGutter=t.pageGutter),t.scrollPaddingBottom!==this.rsProperties.scrollPaddingBottom&&(this.rsProperties.scrollPaddingBottom=t.scrollPaddingBottom),t.scrollPaddingLeft!==this.rsProperties.scrollPaddingLeft&&(this.rsProperties.scrollPaddingLeft=t.scrollPaddingLeft),t.scrollPaddingRight!==this.rsProperties.scrollPaddingRight&&(this.rsProperties.scrollPaddingRight=t.scrollPaddingRight),t.scrollPaddingTop!==this.rsProperties.scrollPaddingTop&&(this.rsProperties.scrollPaddingTop=t.scrollPaddingTop),t.experiments!==this.rsProperties.experiments&&(this.rsProperties.experiments=t.experiments),this.lineLengths.update({fontFace:t.fontFamily,letterSpacing:t.letterSpacing,padding:t.scroll?(t.scrollPaddingLeft||0)+(t.scrollPaddingRight||0):(t.pageGutter||0)*2,wordSpacing:t.wordSpacing,optimalChars:t.optimalLineLength,minChars:t.minimalLineLength,maxChars:t.maximalLineLength});let e=this.updateLayout(t.fontSize,t.deprecatedFontSize||t.iOSPatch,t.scroll,t.columnCount);e?.effectiveContainerWidth&&(this.effectiveContainerWidth=e?.effectiveContainerWidth);let i={a11yNormalize:t.textNormalization,backgroundColor:t.backgroundColor,blendFilter:t.blendFilter,bodyHyphens:typeof t.hyphens!="boolean"?null:t.hyphens?"auto":"none",colCount:e?.colCount,darkenFilter:t.darkenFilter,deprecatedFontSize:t.deprecatedFontSize,fontFamily:t.fontFamily,fontOpticalSizing:typeof t.fontOpticalSizing!="boolean"?null:t.fontOpticalSizing?"auto":"none",fontSize:t.fontSize,fontSizeNormalize:t.fontSizeNormalize,fontWeight:t.fontWeight,fontWidth:t.fontWidth,invertFilter:t.invertFilter,invertGaijiFilter:t.invertGaijiFilter,iOSPatch:t.iOSPatch,iPadOSPatch:t.iPadOSPatch,letterSpacing:t.letterSpacing,ligatures:typeof t.ligatures!="boolean"?null:t.ligatures?"common-ligatures":"none",lineHeight:t.lineHeight,lineLength:e?.effectiveLineLength,linkColor:t.linkColor,noRuby:t.noRuby,paraIndent:t.paragraphIndent,paraSpacing:t.paragraphSpacing,selectionBackgroundColor:t.selectionBackgroundColor,selectionTextColor:t.selectionTextColor,textAlign:t.textAlign,textColor:t.textColor,view:typeof t.scroll!="boolean"?null:t.scroll?"scroll":"paged",visitedColor:t.visitedColor,wordSpacing:t.wordSpacing};this.userProperties=new pe(i)}updateLayout(t,e,i,o){return this.isCJKVertical?this.computeCJKVerticalLength(t,e):i??this.userProperties.view==="scroll"?this.computeScrollLength(t,e):this.paginate(t,e,o)}getCompensatedMetrics(t,e){let i=t||this.userProperties.fontSize||1,o=i<1?1/i:e?i:1;return{zoomFactor:i,zoomCompensation:o,optimal:Math.round(this.lineLengths.optimalLineLength)*i,minimal:this.lineLengths.minimalLineLength!==null?Math.round(this.lineLengths.minimalLineLength*i):null,maximal:this.lineLengths.maximalLineLength!==null?Math.round(this.lineLengths.maximalLineLength*i):null}}paginate(t,e,i){let o=Math.round(Jt(this.containerParent)-this.constraint),n=this.getCompensatedMetrics(t,e),{zoomCompensation:s,optimal:a,minimal:h,maximal:l}=n,d=()=>o>=a&&l!==null?Math.min(Math.round(l*s),o):o,c=1,m=o;if(i===void 0)return{colCount:void 0,effectiveContainerWidth:m,effectiveLineLength:Math.round(m/c*s)};if(i===null)if(o>=a&&l!==null){c=Math.floor(o/a);let p=Math.round(c*(l*s));m=Math.min(p,o)}else m=d();else if(i>1){let p=Math.round(i*(h!==null?h:a));if(o>=p)if(c=i,l===null)m=o;else{let g=Math.round(c*(l*s));m=Math.min(g,o)}else if(h!==null&&o<Math.round(i*h))if(c=Math.floor(o/h),c<=1)c=1,m=d();else{let g=Math.round(c*(a*s));m=Math.min(g,o)}else{c=i;let g=Math.round(c*(a*s));m=Math.min(g,o)}}else c=1,m=d();return{colCount:c,effectiveContainerWidth:m,effectiveLineLength:Math.round(m/c/(t&&t>=1?t:1)*s)}}computeCJKVerticalLength(t,e){let i=Math.round(Jt(this.containerParent)-this.constraint),o=Math.round(Ur(this.containerParent)),n=this.getCompensatedMetrics(t,e),s=n.maximal!==null?Math.min(Math.round(n.maximal*n.zoomCompensation),o):o;return{colCount:void 0,effectiveContainerWidth:i,effectiveLineLength:s}}computeScrollLength(t,e){let i=Math.round(Jt(this.containerParent)-this.constraint),o=this.getCompensatedMetrics(t&&(t<1||e)?t:1,e),n=o.zoomCompensation,s=o.optimal,a=o.maximal,h,l=i,d=Math.round(s*n);if(a===null)d=i;else{let c=Math.min(Math.round(a*n),i);d=e?c:Math.round(c*n)}return{colCount:h,effectiveContainerWidth:l,effectiveLineLength:d}}setContainerWidth(){this.container.style.width=`${this.effectiveContainerWidth}px`}resizeHandler(){let t=this.updateLayout(this.userProperties.fontSize,this.userProperties.deprecatedFontSize||this.userProperties.iOSPatch,this.userProperties.view==="scroll",this.cachedColCount);this.userProperties.colCount=t.colCount,this.userProperties.lineLength=t.effectiveLineLength,this.effectiveContainerWidth=t.effectiveContainerWidth,this.container.style.width=`${this.effectiveContainerWidth}px`}};var Ia="modulepreload",Ha=function(r){return"/"+r},Ir={},et=function(r,t,e){let i=Promise.resolve();if(t&&t.length>0){let n=function(h){return Promise.all(h.map(l=>Promise.resolve(l).then(d=>({status:"fulfilled",value:d}),d=>({status:"rejected",reason:d}))))};document.getElementsByTagName("link");let s=document.querySelector("meta[property=csp-nonce]"),a=s?.nonce||s?.getAttribute("nonce");i=n(t.map(h=>{if(h=Ha(h),h in Ir)return;Ir[h]=!0;let l=h.endsWith(".css"),d=l?'[rel="stylesheet"]':"";if(document.querySelector(`link[href="${h}"]${d}`))return;let c=document.createElement("link");if(c.rel=l?"stylesheet":Ia,l||(c.as="script"),c.crossOrigin="",c.href=h,a&&c.setAttribute("nonce",a),document.head.appendChild(c),l)return new Promise((m,p)=>{c.addEventListener("load",m),c.addEventListener("error",()=>p(new Error(`Unable to preload CSS for ${h}`)))})}))}function o(n){let s=new Event("vite:preloadError",{cancelable:!0});if(s.payload=n,window.dispatchEvent(s),!s.defaultPrevented)throw n}return i.then(n=>{for(let s of n||[])s.status==="rejected"&&o(s.reason);return r().catch(o)})};var Hr=`// Note: we aren't blocking some of the events right now to try and be as nonintrusive as possible.
// For a more comprehensive implementation, see https://github.com/hackademix/noscript/blob/3a83c0e4a506f175e38b0342dad50cdca3eae836/src/content/syncFetchPolicy.js#L142
// The snippet of code at the beginning of this source is an attempt at defence against JS using persistent storage
(function() {
    const noop = () => {}, emptyObj = {}, emptyPromise = () => Promise.resolve(void 0), fakeStorage = {
        getItem: noop, 
        setItem: noop, 
        removeItem: noop, 
        clear: noop, 
        key: noop, 
        length: 0
    };
    
    ["localStorage", "sessionStorage"].forEach((e) => Object.defineProperty(window, e, {
        get: () => fakeStorage,
        configurable: !0
    }));
    
    Object.defineProperty(document, "cookie", {
        get: () => "",
        set: noop,
        configurable: !0
    });
    
    Object.defineProperty(window, "indexedDB", {
        get: () => {},
        configurable: !0
    });
    
    Object.defineProperty(window, "caches", {
        get: () => emptyObj,
        configurable: !0
    });
    
    Object.defineProperty(navigator, "storage", {
        get: () => ({
            persist: emptyPromise,
            persisted: emptyPromise,
            estimate: () => Promise.resolve({quota: 0, usage: 0})
        }),
        configurable: !0
    });
    
    Object.defineProperty(navigator, "serviceWorker", {
        get: () => ({
            register: emptyPromise,
            getRegistration: emptyPromise,
            ready: emptyPromise()
        }),
        configurable: !0
    });

    window._readium_blockedEvents = [];
    window._readium_blockEvents = true;
    window._readium_eventBlocker = (e) => {
        if(!window._readium_blockEvents) return;
        e.preventDefault();
        e.stopImmediatePropagation();
        _readium_blockedEvents.push([
            1, e, e.currentTarget || e.target
        ]);
    };
    window.addEventListener("DOMContentLoaded", window._readium_eventBlocker, true);
    window.addEventListener("load", window._readium_eventBlocker, true);
})();
`;async function us(r,t){let e=r.effectiveLayout===_.fixed,i=t.filter(a=>a.mediaType.isHTML).map(a=>a.href),o=i.length>0?i:[/\.xhtml$/,/\.html$/],n=[{id:"css-selector-generator",as:"script",target:"head",blob:new Blob([ui(wr)],{type:"text/javascript"})},{id:"execution-prevention",as:"script",target:"head",blob:new Blob([ui(Hr)],{type:"text/javascript"}),condition:a=>!!(a.querySelector("script")||a.querySelector("body[onload]:not(body[onload=''])"))}],s=[{id:"onload-proxy",as:"script",target:"head",blob:new Blob([ui(Cr)],{type:"text/javascript"}),condition:a=>!!(a.querySelector("script")||a.querySelector("body[onload]:not(body[onload=''])"))}];if(!e){let a=Ft(r),h,l,d;switch(a){case"rtl":{let[c,m,p]=await Promise.all([et(()=>Promise.resolve().then(()=>(Br(),Wr)),[]),et(()=>Promise.resolve().then(()=>(Gr(),jr)),[]),et(()=>Promise.resolve().then(()=>(Xr(),Vr)),[])]);h=c.default,l=m.default,d=p.default;break}case"cjk-horizontal":{let[c,m,p]=await Promise.all([et(()=>Promise.resolve().then(()=>(Kr(),$r)),[]),et(()=>Promise.resolve().then(()=>(qr(),Yr)),[]),et(()=>Promise.resolve().then(()=>(Zr(),Jr)),[])]);h=c.default,l=m.default,d=p.default;break}case"cjk-vertical":case"mongolian-vertical":{let[c,m,p]=await Promise.all([et(()=>Promise.resolve().then(()=>(ts(),Qr)),[]),et(()=>Promise.resolve().then(()=>(is(),es)),[]),et(()=>Promise.resolve().then(()=>(ns(),os)),[])]);h=c.default,l=m.default,d=p.default;break}default:{let[c,m,p]=await Promise.all([et(()=>Promise.resolve().then(()=>(ss(),rs)),[]),et(()=>Promise.resolve().then(()=>(ls(),as)),[]),et(()=>Promise.resolve().then(()=>(cs(),hs)),[])]);h=c.default,l=m.default,d=p.default;break}}if(n.unshift({id:"readium-css-before",as:"link",target:"head",blob:new Blob([ke(h)],{type:"text/css"}),rel:"stylesheet"}),s.unshift({id:"readium-css-default",as:"link",target:"head",blob:new Blob([ke(l)],{type:"text/css"}),rel:"stylesheet",condition:c=>!(c.querySelector("link[rel='stylesheet']")||c.querySelector("style")||c.querySelector("[style]:not([style=''])"))},{id:"readium-css-after",as:"link",target:"head",blob:new Blob([ke(d)],{type:"text/css"}),rel:"stylesheet"}),(a==="cjk-horizontal"||a==="cjk-vertical")&&(r.description==="ebpaj-guide-1.0"||r.otherMetadata?.["ebpaj:guide-version"]!==void 0)){let{default:c}=await et(async()=>{let{default:m}=await Promise.resolve().then(()=>(ms(),ds));return{default:m}},[]);s.push({id:"readium-css-ebpaj",as:"link",target:"head",blob:new Blob([ke(c)],{type:"text/css"}),rel:"stylesheet"})}}return[{resources:o,prepend:n,append:s}]}var ze=class{constructor(t,e){this.backgroundColor=t.backgroundColor||e.backgroundColor||null,this.blendFilter=typeof t.blendFilter=="boolean"?t.blendFilter:e.blendFilter??null,this.columnCount=t.columnCount!==void 0?t.columnCount:e.columnCount!==void 0?e.columnCount:null,this.constraint=t.constraint||e.constraint,this.darkenFilter=typeof t.darkenFilter=="boolean"?t.darkenFilter:e.darkenFilter??null,this.deprecatedFontSize=typeof t.deprecatedFontSize=="boolean"?t.deprecatedFontSize:e.deprecatedFontSize??null,this.fontFamily=t.fontFamily||e.fontFamily||null,this.fontSize=t.fontSize!==void 0?t.fontSize:e.fontSize!==void 0?e.fontSize:null,this.fontSizeNormalize=typeof t.fontSizeNormalize=="boolean"?t.fontSizeNormalize:e.fontSizeNormalize??null,this.fontOpticalSizing=typeof t.fontOpticalSizing=="boolean"?t.fontOpticalSizing:e.fontOpticalSizing??null,this.fontWeight=t.fontWeight!==void 0?t.fontWeight:e.fontWeight!==void 0?e.fontWeight:null,this.fontWidth=t.fontWidth!==void 0?t.fontWidth:e.fontWidth!==void 0?e.fontWidth:null,this.hyphens=typeof t.hyphens=="boolean"?t.hyphens:e.hyphens??null,this.invertFilter=typeof t.invertFilter=="boolean"?t.invertFilter:e.invertFilter??null,this.invertGaijiFilter=typeof t.invertGaijiFilter=="boolean"?t.invertGaijiFilter:e.invertGaijiFilter??null,this.iOSPatch=this.deprecatedFontSize||t.iOSPatch===!1?!1:t.iOSPatch===!0?(st.OS.iOS||st.OS.iPadOS)&&st.iOSRequest==="mobile":e.iOSPatch,this.iPadOSPatch=this.deprecatedFontSize||t.iPadOSPatch===!1?!1:t.iPadOSPatch===!0?st.OS.iPadOS&&st.iOSRequest==="desktop":e.iPadOSPatch,this.letterSpacing=t.letterSpacing!==void 0?t.letterSpacing:e.letterSpacing!==void 0?e.letterSpacing:null,this.ligatures=typeof t.ligatures=="boolean"?t.ligatures:e.ligatures??null,this.lineHeight=t.lineHeight!==void 0?t.lineHeight:e.lineHeight!==void 0?e.lineHeight:null,this.linkColor=t.linkColor||e.linkColor||null,this.maximalLineLength=t.maximalLineLength===null?null:t.maximalLineLength||e.maximalLineLength||null,this.minimalLineLength=t.minimalLineLength===null?null:t.minimalLineLength||e.minimalLineLength||null,this.noRuby=typeof t.noRuby=="boolean"?t.noRuby:e.noRuby??null,this.optimalLineLength=t.optimalLineLength||e.optimalLineLength,this.pageGutter=t.pageGutter!==void 0?t.pageGutter:e.pageGutter!==void 0?e.pageGutter:null,this.paragraphIndent=t.paragraphIndent!==void 0?t.paragraphIndent:e.paragraphIndent!==void 0?e.paragraphIndent:null,this.paragraphSpacing=t.paragraphSpacing!==void 0?t.paragraphSpacing:e.paragraphSpacing!==void 0?e.paragraphSpacing:null,this.scroll=typeof t.scroll=="boolean"?t.scroll:e.scroll??null,this.scrollPaddingTop=t.scrollPaddingTop!==void 0?t.scrollPaddingTop:e.scrollPaddingTop!==void 0?e.scrollPaddingTop:null,this.scrollPaddingBottom=t.scrollPaddingBottom!==void 0?t.scrollPaddingBottom:e.scrollPaddingBottom!==void 0?e.scrollPaddingBottom:null,this.scrollPaddingLeft=t.scrollPaddingLeft!==void 0?t.scrollPaddingLeft:e.scrollPaddingLeft!==void 0?e.scrollPaddingLeft:null,this.scrollPaddingRight=t.scrollPaddingRight!==void 0?t.scrollPaddingRight:e.scrollPaddingRight!==void 0?e.scrollPaddingRight:null,this.selectionBackgroundColor=t.selectionBackgroundColor||e.selectionBackgroundColor||null,this.selectionTextColor=t.selectionTextColor||e.selectionTextColor||null,this.textAlign=t.textAlign||e.textAlign||null,this.textColor=t.textColor||e.textColor||null,this.textNormalization=typeof t.textNormalization=="boolean"?t.textNormalization:e.textNormalization??null,this.visitedColor=t.visitedColor||e.visitedColor||null,this.wordSpacing=t.wordSpacing!==void 0?t.wordSpacing:e.wordSpacing!==void 0?e.wordSpacing:null,this.experiments=e.experiments||null}};var tl="'Iowan Old Style', Sitka, 'Sitka Text', Palatino, 'Book Antiqua', 'URW Palladio L', P052, serif",ps={oldStyleTf:tl};var el=16,gs=ps.oldStyleTf,Ni=class r{constructor(t){this._optimalLineLength=null,this._canvas=document.createElement("canvas"),this._optimalChars=t.optimalChars,this._minChars=t.minChars,this._maxChars=t.maxChars,this._baseFontSize=t.baseFontSize||el,this._fontFace=t.fontFace||gs,this._sample=t.sample||null,this._padding=t.padding??0,this._letterSpacing=t.letterSpacing?Math.round(t.letterSpacing*this._baseFontSize):0,this._wordSpacing=t.wordSpacing?Math.round(t.wordSpacing*this._baseFontSize):0,this._isCJK=t.isCJK||!1,this._getRelative=t.getRelative||!1,this._minDivider=this._minChars&&this._minChars<this._optimalChars?this._optimalChars/this._minChars:this._minChars===null?null:1,this._maxMultiplier=this._maxChars&&this._maxChars>this._optimalChars?this._maxChars/this._optimalChars:this._maxChars===null?null:1,this._approximatedWordSpaces=r.approximateWordSpaces(this._optimalChars,this._sample)}updateMultipliers(){this._minDivider=this._minChars&&this._minChars<this._optimalChars?this._optimalChars/this._minChars:this._minChars===null?null:1,this._maxMultiplier=this._maxChars&&this._maxChars>this._optimalChars?this._maxChars/this._optimalChars:this._maxChars===null?null:1}update(t){t.optimalChars&&(this._optimalChars=t.optimalChars),t.minChars!==void 0&&(this._minChars=t.minChars),t.maxChars!==void 0&&(this._maxChars=t.maxChars),t.baseFontSize&&(this._baseFontSize=t.baseFontSize),t.fontFace!==void 0&&(this._fontFace=t.fontFace||gs),t.letterSpacing&&(this._letterSpacing=t.letterSpacing),t.wordSpacing&&(this._wordSpacing=t.wordSpacing),t.isCJK!=null&&(this._isCJK=t.isCJK),t.padding!==void 0&&(this._padding=t.padding??0),t.getRelative&&(this._getRelative=t.getRelative),t.sample&&(this._sample=t.sample,this._approximatedWordSpaces=r.approximateWordSpaces(this._optimalChars,this._sample)),this.updateMultipliers(),this._optimalLineLength=this.getOptimalLineLength()}get baseFontSize(){return this._baseFontSize}get minimalLineLength(){return this._optimalLineLength||(this._optimalLineLength=this.getOptimalLineLength()),this._minDivider!==null?Math.round(this._optimalLineLength/this._minDivider+this._padding)/(this._getRelative?this._baseFontSize:1):null}get maximalLineLength(){return this._optimalLineLength||(this._optimalLineLength=this.getOptimalLineLength()),this._maxMultiplier!==null?Math.round(this._optimalLineLength*this._maxMultiplier+this._padding)/(this._getRelative?this._baseFontSize:1):null}get optimalLineLength(){return this._optimalLineLength||(this._optimalLineLength=this.getOptimalLineLength()),Math.round(this._optimalLineLength+this._padding)/(this._getRelative?this._baseFontSize:1)}get all(){return this._optimalLineLength||(this._optimalLineLength=this.getOptimalLineLength()),{min:this.minimalLineLength,max:this.maximalLineLength,optimal:this.optimalLineLength,baseFontSize:this._baseFontSize}}static approximateWordSpaces(t,e){let i=0;if(e&&e.length>=t){let o=e.match(/([\s]+)/gi);i=(o?o.length:0)*(t/e.length)}return i}getLineLengthFallback(){let t=this._letterSpacing*(this._optimalChars-1),e=this._wordSpacing*this._approximatedWordSpaces;return this._optimalChars*(this._baseFontSize*.5)+t+e}getOptimalLineLength(){if(this._fontFace){if(typeof this._fontFace=="string")return this.measureText(this._fontFace);{let t=new FontFace(this._fontFace.name,`url(${this._fontFace.url})`);t.load().then(()=>(document.fonts.add(t),this.measureText(t.family)),e=>{})}}return this.getLineLengthFallback()}measureText(t){let e=this._canvas.getContext("2d");if(e&&t){let i=this._isCJK?"\u6C34".repeat(this._optimalChars):"0".repeat(this._optimalChars);if(e.font=`${this._baseFontSize}px ${t}`,this._sample&&this._sample.length>=this._optimalChars&&(i=this._sample.slice(0,this._optimalChars)),Object.hasOwn(e,"letterSpacing")&&Object.hasOwn(e,"wordSpacing"))return e.letterSpacing=this._letterSpacing.toString()+"px",e.wordSpacing=this._wordSpacing.toString()+"px",e.measureText(i).width;{let o=this._letterSpacing*(this._optimalChars-1),n=this._wordSpacing*r.approximateWordSpaces(this._optimalChars,this._sample);return e.measureText(i).width+o+n}}else return this.getLineLengthFallback()}};var il=1e3,ol=r=>({frameLoaded:r.frameLoaded||(()=>{}),positionChanged:r.positionChanged||(()=>{}),viewportChanged:r.viewportChanged||(()=>{}),timelineItemChanged:r.timelineItemChanged||(()=>{}),tap:r.tap||(()=>!1),click:r.click||(()=>!1),zoom:r.zoom||(()=>{}),miscPointer:r.miscPointer||(()=>{}),scroll:r.scroll||(()=>{}),customEvent:r.customEvent||(()=>{}),handleLocator:r.handleLocator||(()=>!1),textSelected:r.textSelected||(()=>{}),contentProtection:r.contentProtection||(()=>{}),contextMenu:r.contextMenu||(()=>{}),peripheral:r.peripheral||(()=>{})});function nl(r,t){return r.length===t.length&&r.every((e,i)=>e===t[i])}var Ai=class r extends si{constructor(t,e,i,o=[],n=void 0,s={preferences:{},defaults:{}}){super(),this._destroyed=!1,this._visibleFragmentIds=[],this._notifiedVisibleFragmentIds=[],this._timelineAugmented=!1,this._preferencesEditor=null,this._injector=null,this._isNavigating=!1,this._navigatorProtector=null,this._keyboardPeripheralsManager=null,this._suspiciousActivityListener=null,this._keyboardPeripheralListener=null,this._decorations=new Map,this._decorationObservers=new Map,this._decorationHoveredDecorations=new Map,this._decorationActivationState=new Map,this._decorationHoverState=new Map,this._decorationActivationConsumed=!1,this._navigationLocked=!1,this._pointedPieceRequests=new Map,this._pointedPieceRequestId=0,this.reflowViewport={readingOrder:[],progressions:new Map,positions:null},this.pub=e,this.container=t,this.listeners=ol(i),this.currentLocation=n,o.length&&(this.positions=o),this._preferences=new Yt(s.preferences),this._defaults=new Li(s.defaults),this._settings=new ze(this._preferences,this._defaults);let a=Ft(e.metadata),h=a==="cjk-horizontal",l=a==="cjk-vertical",d=l||a==="mongolian-vertical",c=h||l;this._css=new Mi({rsProperties:new Oi({noVerticalPagination:d||void 0}),userProperties:new pe({}),lineLengths:new Ni({optimalChars:this._settings.optimalLineLength,minChars:this._settings.minimalLineLength,maxChars:this._settings.maximalLineLength,padding:this._settings.scroll?(this._settings.scrollPaddingLeft||0)+(this._settings.scrollPaddingRight||0):(this._settings.pageGutter||0)*2,fontFace:this._settings.fontFamily,letterSpacing:this._settings.letterSpacing,wordSpacing:this._settings.wordSpacing,isCJK:c}),container:t,constraint:this._settings.constraint,isCJKVertical:d}),this._layout=r.determineLayout(e,!!this._settings.scroll),this.currentProgression=e.metadata.effectiveReadingProgression,this._injectablesConfig=s.injectables||{rules:[],allowedDomains:[]},this._readiumRulesPromise=us(e.metadata,e.readingOrder.items),this._contentProtection=s.contentProtection||{},this._decoratorConfig=s.decoratorConfig||{},this._decorationResizeSelectors=new Set(this._decoratorConfig.resizeWatchSelectors??[]),this._upgradeInsecureRequests=s.upgradeInsecureRequests??"always",this._keyboardPeripherals=this.mergeKeyboardPeripherals(this._contentProtection,s.keyboardPeripherals||[]),(this._contentProtection.disableContextMenu||this._contentProtection.checkAutomation||this._contentProtection.checkIFrameEmbedding||this._contentProtection.monitorDevTools||this._contentProtection.protectPrinting?.disable)&&(this._navigatorProtector=new _i(this._contentProtection),this._suspiciousActivityListener=m=>{let{type:p,...g}=m.detail;p==="context_menu"?this.listeners.contextMenu(g):this.listeners.contentProtection(p,g)},window.addEventListener(bi,this._suspiciousActivityListener)),this._keyboardPeripherals.length>0&&(this._keyboardPeripheralsManager=new vi({keyboardPeripherals:this._keyboardPeripherals}),this._keyboardPeripheralListener=m=>{let p=m.detail;this.listeners.peripheral(p)},window.addEventListener(Ri,this._keyboardPeripheralListener)),this.resizeObserver=new ResizeObserver(()=>this.ownerWindow.requestAnimationFrame(async()=>await this.resizeHandler())),this.resizeObserver.observe(this.container.parentElement||document.documentElement)}static determineLayout(t,e){let i=t.metadata.effectiveLayout;if(i===_.fixed)return _.fixed;if(t.metadata.otherMetadata&&"http://openmangaformat.org/schema/1.0#version"in t.metadata.otherMetadata)return _.fixed;if(t.metadata?.conformsTo?.includes(Z.DIVINA))return _.fixed;if(i===_.scrolled)return _.scrolled;let o=Ft(t.metadata);return o==="cjk-vertical"||o==="mongolian-vertical"?_.scrolled:i===_.reflowable&&e?_.scrolled:_.reflowable}async load(){if(this.positions?.length||(this.positions=await this.pub.positionsFromManifest()),!this._injector){let t=await this._readiumRulesPromise;this._injector=new mi({rules:[...t,...this._injectablesConfig.rules],allowedDomains:this._injectablesConfig.allowedDomains})}if(!this._destroyed){if(this._layout===_.fixed)this.framePool=new zi(this.container,this.positions,this.pub,this._layout,this._injector,this._contentProtection,this._keyboardPeripherals,[...this._decorationResizeSelectors],this._upgradeInsecureRequests),this.framePool.listener=(t,e)=>{this.eventListener(t,e)};else{if(await this.updateCSS(!1),this._destroyed)return;let t=this.compileCSSProperties(this._css);this.framePool=new wi(this.container,this.positions,t,this._injector,this._contentProtection,this._keyboardPeripherals,e=>this.pub.timeline.segmentsForHref(e).flatMap(i=>i.references).map(i=>{let o=i.indexOf("#");return o>=0?i.slice(o+1):""}).filter(Boolean),[...this._decorationResizeSelectors],this._upgradeInsecureRequests)}if(this.currentLocation===void 0?this.currentLocation=this.positions[0]:this.currentLocation=this.completeLocator(this.currentLocation),await this.resizeHandler(),!this._destroyed)return new Promise(t=>{if(this._destroyed)return t(!1);this.go(this.currentLocation,!1,t)})}}get settings(){if(this._layout===_.fixed)return Object.freeze({...this._settings});{let t=this._css.userProperties.colCount||this._css.rsProperties.colCount||this._settings.columnCount;return Object.freeze({...this._settings,columnCount:t})}}get preferencesEditor(){return this._preferencesEditor===null&&(this._preferencesEditor=new Te(this._preferences,this.settings,this.pub.metadata)),this._preferencesEditor}async submitPreferences(t){this._preferences=this._preferences.merging(t),await this.applyPreferences()}async applyPreferences(){let t=this._settings;this._settings=new ze(this._preferences,this._defaults),this._preferencesEditor!==null&&(this._preferencesEditor=new Te(this._preferences,this.settings,this.pub.metadata)),this._layout===_.fixed?this.handleFXLPrefs(t,this._settings):await this.updateCSS(!0)}handleFXLPrefs(t,e){t.columnCount!==e.columnCount&&(this.framePool.setPerPage(e.columnCount),this.ownerWindow.requestAnimationFrame(()=>this.notifyViewportChange(this.listeners.viewportChanged)))}async updateCSS(t){this._css.update(this._settings),t&&await this.commitCSS(this._css)}compileCSSProperties(t){let e={};for(let[i,o]of Object.entries(t.rsProperties.toCSSProperties()))e[i]=o;for(let[i,o]of Object.entries(t.userProperties.toCSSProperties()))e[i]=o;return e}async commitCSS(t){if(!this.framePool)return;let e=this.compileCSSProperties(t);this.framePool.setCSSProperties(e),this._css.userProperties.view==="paged"&&this._layout===_.scrolled?await this.setLayout(_.reflowable):this._css.userProperties.view==="scroll"&&this._layout===_.reflowable&&await this.setLayout(_.scrolled),this._css.setContainerWidth()}async resizeHandler(){let t=this.container.parentElement||document.documentElement;if(this._layout===_.fixed){let e=Jt(t)-this._settings.constraint;if(mt()?this.container.attributeStyleMap.set("width",CSS.px(e)):this.container.style.width=`${e}px`,!this.framePool)return;this.framePool.resizeHandler(),this.notifyViewportChange(this.listeners.viewportChanged)}else{let e=this._css.userProperties.colCount,i=this._css.userProperties.lineLength;this._css.resizeHandler(),(this._css.userProperties.view!=="scroll"&&e!==this._css.userProperties.colCount||i!==this._css.userProperties.lineLength)&&await this.commitCSS(this._css)}}get layout(){return this._layout}get ownerWindow(){return this.container.ownerDocument.defaultView||window}get _cframes(){return(this.framePool?.currentFrames??[]).filter(t=>!(t instanceof gt&&t.isDestroyed))}get pool(){return this.framePool}eventListener(t,e,i){switch(t){case"_pong":if(this.listeners.frameLoaded(this._cframes[0].iframe.contentWindow),this.listeners.positionChanged(this.currentLocation),this._notifyTimelineChange(this.currentLocation),i){let l=this._cframes.filter(c=>!!c).indexOf(i),d=l>=0?this.viewport.readingOrder[l]:void 0;d&&this._reapplyDecorationsToFrame(i,d)}else this._reapplyDecorationsToCurrentFrames();this._watchTextLayoutInCurrentFrame(!0),this._sendNavigationLock();break;case"text_layout":i&&i===this._textLayoutFrame&&this.currentLocation.href.split("#")[0]===this._textLayoutWatch?.href&&this._textLayoutWatch.cb(e);break;case"pointed_piece":{let{id:l,index:d}=e;this._pointedPieceRequests.get(l)?.(d);break}case"first_visible_locator":let o=z.deserialize(e);if(!o)break;this.currentLocation=new z({href:this.currentLocation.href,type:this.currentLocation.type,title:this.currentLocation.title,locations:o?.locations,text:o?.text}),this.listeners.positionChanged(this.currentLocation),this._notifyTimelineChange(this.currentLocation);break;case"text_selected":{let l=e;if(i){let d=this._cframes.filter(m=>!!m).indexOf(i),c=d>=0?this.viewport.readingOrder[d]:void 0;if(c){let m=this.pub.readingOrder.findWithHref(c);l.locator=new z({href:c,type:m.type||"application/xhtml+xml",text:new at({highlight:l.text})})}}this.listeners.textSelected(l);break}case"decoration_activated":{this._handleDecorationActivated(e)&&(this._decorationActivationConsumed=!0);break}case"decoration_pointer_enter":this._handleDecorationPointerEnter(e);break;case"decoration_pointer_leave":this._handleDecorationPointerLeave(e);break;case"click":case"tap":if(this._decorationActivationConsumed){this._decorationActivationConsumed=!1;break}let n=e;if(n.interactiveElement){let l=new DOMParser().parseFromString(n.interactiveElement,"text/html").body.children[0];if(l.nodeType===l.ELEMENT_NODE&&l.nodeName==="A"&&l.hasAttribute("href")){let d=l.attributes.getNamedItem("href")?.value;if(d.startsWith("#"))this._navigationLocked||this.go(this.currentLocation.copyWithLocations({fragments:[d.substring(1)]}),!1,()=>{});else if(d.startsWith("http://")||d.startsWith("https://")||d.startsWith("mailto:")||d.startsWith("tel:"))this.listeners.handleLocator(new q({href:d}).locator);else if(!this._navigationLocked)try{this.goLink(new q({href:zo.join(zo.dirname(this.currentLocation.href),d)}),!1,()=>{})}catch(c){console.warn(`Couldn't go to link for ${d}: ${c}`),this.listeners.handleLocator(new q({href:d}).locator)}}else console.log("Clicked on",l)}else{if(this._layout===_.fixed&&this.framePool.doNotDisturb&&(n.doNotDisturb=!0),n.x+=this._spreadOffsetX(n.targetFrameSrc),t==="click"?this.listeners.click(n):this.listeners.tap(n))break;let l=(this._cframes.length===2?this._cframes[0].window.innerWidth+this._cframes[1].window.innerWidth:this._cframes[0].window.innerWidth)*window.devicePixelRatio/4;if(n.x>=l&&n.x<=l*3&&this.listeners.miscPointer(1),this._navigationLocked)break;n.x<l?this.goLeft(!1,()=>{}):n.x>l*3&&this.goRight(!1,()=>{})}break;case"tap_more":this.listeners.miscPointer(e);break;case"no_more":this._navigationLocked||this.changeResource(1);break;case"no_less":this._navigationLocked||this.changeResource(-1);break;case"swipe":break;case"scroll":this.listeners.scroll(e);break;case"zoom":this.listeners.zoom(e);break;case"progress":{let l=i instanceof gt?this.framePool._frameFor(this.currentLocation.href.split("#")[0]):void 0;if(i instanceof gt&&i!==l)break;this.syncLocation(e);break}case"content_protection":let s=e;this.listeners.contentProtection(s.type,s);break;case"context_menu":this.listeners.contextMenu(e);break;case"keyboard_peripherals":let a=e,h={...a,interactiveElement:void 0};a.interactiveElement&&(h.interactiveElement=new DOMParser().parseFromString(a.interactiveElement,"text/html").body.children[0]),this.listeners.peripheral(h);break;case"log":console.log(this._cframes[0]?.source?.split("/")[3],...e);break;default:this.listeners.customEvent(t,e);break}}determineModules(){let t=Array.from(me.keys());if(this._layout===_.fixed)return t.filter(o=>vo.includes(o));t=t.filter(o=>Ro.includes(o));let e=Ft(this.pub.metadata);if(e==="cjk-vertical"||e==="mongolian-vertical")return t.filter(o=>o!=="column_snapper"&&o!=="scroll_snapper");let i=t;return this._layout===_.scrolled?t=i.filter(o=>o!=="column_snapper"&&o!=="cjk_vertical_snapper"):t=i.filter(o=>o!=="scroll_snapper"&&o!=="cjk_vertical_snapper"),t}attachListener(){let t=this._cframes.filter(e=>!!e);if(t.length===0)throw Error("no cframe to attach listener to");t.forEach(e=>{e.msg&&(e.msg.listener=(i,o)=>{this.eventListener(i,o,e)})}),this._reapplyDecorationsToCurrentFrames(),this._watchTextLayoutInCurrentFrame(),this._sendNavigationLock()}async apply(){if(await this.framePool.update(this.pub,this.currentLocator,this.determineModules()),this.attachListener(),this.pub.readingOrder.findIndexWithHref(this.currentLocation.href)<0)throw Error("Link for "+this.currentLocation.href+" not found!");this.notifyViewportChange(this.listeners.viewportChanged)}supportsDecorationStyle(t){return xo(t,this._decoratorConfig.decorationTemplates)}registerDecorationObserver(t,e){this._decorationObservers.has(t)||this._decorationObservers.set(t,new Set),this._decorationObservers.get(t).add(e),e.onDecorationActivated&&(this._decorationActivationState.set(t,!0),this._sendDecorationActivationToFrames(t,!0)),(e.onDecorationPointerEnter||e.onDecorationPointerLeave)&&(this._decorationHoverState.set(t,!0),this._sendDecorationHoverToFrames(t,!0))}unregisterDecorationObserver(t){this._decorationObservers.forEach((e,i)=>{if(!e.has(t))return;e.delete(t);let o=[...e].some(s=>s.onDecorationActivated);this._decorationActivationState.has(i)&&!o&&(this._decorationActivationState.delete(i),this._sendDecorationActivationToFrames(i,!1));let n=[...e].some(s=>s.onDecorationPointerEnter||s.onDecorationPointerLeave);this._decorationHoverState.has(i)&&!n&&(this._decorationHoverState.delete(i),this._sendDecorationHoverToFrames(i,!1))})}_sendDecorationActivationToFrames(t,e){this._cframes.filter(i=>!!i).forEach(i=>{i.msg&&i.msg.send("decoration_activatable",{group:t,activatable:e})})}_sendDecorationHoverToFrames(t,e){this._cframes.filter(i=>!!i).forEach(i=>{i.msg&&i.msg.send("decoration_hoverable",{group:t,hoverable:e})})}addDecorationResizeTarget(t){this._decorationResizeSelectors.add(t),this.framePool?.addResizeTarget(t)}removeDecorationResizeTarget(t){this._decorationResizeSelectors.delete(t),this.framePool?.removeResizeTarget(t)}applyDecorations(t,e){let i=this._decorations.get(e)??[],o=new Map(i.map(c=>[c.id,c])),n=new Map(t.map(c=>[c.id,c])),s=[],a=[],h=[];for(let[c,m]of o)n.has(c)?ko(m,n.get(c))||h.push(n.get(c)):s.push(c);for(let[c,m]of n)o.has(c)||a.push(m);this._decorations.set(e,t),this._sendDecorationOps(e,s,a,h,i);let l=this._decorationActivationState.get(e);l!==void 0&&this._sendDecorationActivationToFrames(e,l);let d=this._decorationHoverState.get(e);d!==void 0&&this._sendDecorationHoverToFrames(e,d)}_sendDecorationOps(t,e,i,o,n){let s=this._cframes.filter(l=>!!l),a=new Map(n.map(l=>[l.id,l])),h=this.viewport.readingOrder;s.forEach((l,d)=>{if(!l.msg)return;let c=h[d];if(c){for(let m of e){let p=a.get(m);!p||p.locator.href!==c||l.msg.send("decorate",{group:t,action:"remove",decoration:{id:m}})}for(let m of i)m.locator.href===c&&l.msg.send("decorate",{group:t,action:"add",decoration:Fe(m,this._decoratorConfig.decorationTemplates)});for(let m of o)a.get(m.id)?.locator.href===c&&m.locator.href!==c&&l.msg.send("decorate",{group:t,action:"remove",decoration:{id:m.id}}),m.locator.href===c&&l.msg.send("decorate",{group:t,action:"update",decoration:Fe(m,this._decoratorConfig.decorationTemplates)})}})}_reapplyDecorationsToFrame(t,e){if(t.msg){for(let[i,o]of this._decorations){let n=o.filter(s=>s.locator.href===e);t.msg.send("decorate",{group:i,action:"clear"});for(let s of n)t.msg.send("decorate",{group:i,action:"add",decoration:Fe(s,this._decoratorConfig.decorationTemplates)})}for(let[i,o]of this._decorationActivationState)t.msg.send("decoration_activatable",{group:i,activatable:o});for(let[i,o]of this._decorationHoverState)t.msg.send("decoration_hoverable",{group:i,hoverable:o})}}_reapplyDecorationsToCurrentFrames(){let t=this._cframes.filter(i=>!!i),e=this.viewport.readingOrder;t.forEach((i,o)=>{let n=e[o];n&&this._reapplyDecorationsToFrame(i,n)})}watchTextLayout(t,e){let i={href:t[0]?.href.split("#")[0]??"",pieces:t.map(o=>[o.text?.serialize(),_t(o.locations)]),cb:e};return this._textLayoutWatch=i,this._watchTextLayoutInCurrentFrame(!0),()=>{this._textLayoutWatch===i&&(this._textLayoutWatch=void 0,this._watchTextLayoutInCurrentFrame())}}lockNavigation(t){this._navigationLocked=t,this._sendNavigationLock()}_sendNavigationLock(){this._cframes.forEach(t=>{t instanceof gt&&t.msg?.send("lock_navigation",this._navigationLocked)})}_spreadOffsetX(t){if(this._layout!==_.fixed||this.currentProgression!==E.rtl&&this.currentProgression!==E.ltr)return 0;let e=this.framePool.currentFrames;if(e.length<2)return 0;let i=this.currentProgression===E.rtl;return t!==e[i?0:1]?.source?0:(e[i?1:0]?.iframe.contentWindow?.innerWidth??0)*window.devicePixelRatio}async findPointedPiece(t,e){let i=this._cframes.filter(d=>!!d),o=i.findIndex(d=>d.source===t.targetFrameSrc),n=o>=0?this.viewport.readingOrder[o]:void 0;if(!n||!t.cssSelector)return;let s=e.filter(d=>d.href.split("#")[0]===n);if(s.length===0)return;let a=`${++this._pointedPieceRequestId}`,h,l=await new Promise(d=>{this._pointedPieceRequests.set(a,d),h=setTimeout(()=>d(-1),il);let c={id:a,cssSelector:t.cssSelector,x:t.x-this._spreadOffsetX(t.targetFrameSrc),y:t.y,pieces:s.map(m=>[m.text?.serialize(),_t(m.locations)])};i[o].msg?.send("pointed_piece",c,()=>d(-1))||d(-1)});return clearTimeout(h),this._pointedPieceRequests.delete(a),s[l]}_watchTextLayoutInCurrentFrame(t=!1){let e=this._cframes[0],i=this._textLayoutWatch,o=i&&e instanceof gt&&this.currentLocation?.href.split("#")[0]===i.href?e:void 0;if(o===this._textLayoutFrame&&!t)return;let n=this._textLayoutFrame;n&&n!==o&&!n.isDestroyed&&n.msg?.send("watch_text_layout",[]),this._textLayoutFrame=o,o&&i&&o.msg?.send("watch_text_layout",i.pieces)}_handleDecorationActivated(t){let e=this._decorationObservers.get(t.group);if(!e||e.size===0)return!1;let i=(this._decorations.get(t.group)??[]).find(s=>s.id===t.decorationId);if(!i)return!1;let o={decoration:i,group:t.group,rect:t.rect,point:t.point},n=!1;for(let s of e)s.onDecorationActivated?.(o)&&(n=!0);return n}_handleDecorationPointerEnter(t){let e=this._decorationObservers.get(t.group);if(!e||e.size===0)return;let i=(this._decorations.get(t.group)??[]).find(n=>n.id===t.decorationId);if(!i)return;this._decorationHoveredDecorations.set(t.group,i);let o={decoration:i,group:t.group,rect:t.rect,point:t.point};for(let n of e)n.onDecorationPointerEnter?.(o)}_handleDecorationPointerLeave(t){let e=this._decorationObservers.get(t.group);if(!e||e.size===0)return;let i=(this._decorations.get(t.group)??[]).find(n=>n.id===t.decorationId)??this._decorationHoveredDecorations.get(t.group);if(this._decorationHoveredDecorations.delete(t.group),!i)return;let o={decoration:i,group:t.group,rect:t.rect,point:t.point};for(let n of e)n.onDecorationPointerLeave?.(o)}async destroy(){this._destroyed=!0,this._suspiciousActivityListener&&window.removeEventListener(bi,this._suspiciousActivityListener),this._keyboardPeripheralListener&&window.removeEventListener(Ri,this._keyboardPeripheralListener),this._navigatorProtector?.destroy(),this._keyboardPeripheralsManager?.destroy(),await this.framePool?.destroy(),this._decorations.clear(),this._decorationObservers.clear(),this._decorationHoveredDecorations.clear(),this._decorationActivationState.clear(),this._decorationHoverState.clear(),this._decorationResizeSelectors.clear()}async changeResource(t){if(t===0)return!1;if(this._layout===_.fixed){let o=this.framePool,n=o.viewport.positions[0];if(t===1){if(!o.next(o.perPage))return!1}else if(t===-1){if(!o.prev(o.perPage))return!1}else throw Error("Invalid relative value for FXL");let s=o.viewport.positions[0];if(n>s){for(let a=this.positions.length-1;a>=0;a--)if(this.positions[a].href===this.pub.readingOrder.items[s-1].href){this.currentLocation=this.positions[a].copyWithLocations({progression:.999999999999});break}}else if(n<s){for(let a=0;a<this.positions.length;a++)if(this.positions[a].href===this.pub.readingOrder.items[s-1].href){this.currentLocation=this.positions[a];break}}return await this.apply(),this.listeners.positionChanged(this.currentLocation),this._notifyTimelineChange(this.currentLocation),!0}let e=this.pub.readingOrder.findIndexWithHref(this.currentLocation.href),i=Math.max(0,Math.min(this.pub.readingOrder.items.length-1,e+t));if(i===e)return this._cframes[0]?.msg?.send("shake",void 0,async o=>{}),!1;if(e>i){for(let o=this.positions.length-1;o>=0;o--)if(this.positions[o].href===this.pub.readingOrder.items[i].href){this.currentLocation=this.positions[o].copyWithLocations({progression:.999999999999});break}}else for(let o=0;o<this.positions.length;o++)if(this.positions[o].href===this.pub.readingOrder.items[i].href){this.currentLocation=this.positions[o];break}return await this.apply(),!0}findLastPositionInProgressionRange(t,e){let i=t.findLastIndex(o=>{let n=o.locations.progression;return!!(n&&n>e.start&&n<=e.end)});return i!==-1?t[i]:void 0}findNearestPositions(t){let e=this.positions.filter(s=>s.href===this.currentLocation.href),i=this.currentLocation,o,n=e.findLastIndex(s=>(s.locations.progression??0)<=t.start);if(n!==-1){i=e[n];let s=e.slice(n+1);o=this.findLastPositionInProgressionRange(s,t)}return{first:i,last:o}}updateViewport(t){this.reflowViewport.readingOrder=[],this.reflowViewport.progressions.clear(),this.reflowViewport.positions=null,this.currentLocation&&(this.reflowViewport.readingOrder.push(this.currentLocation.href),this.reflowViewport.progressions.set(this.currentLocation.href,t),this.currentLocation.locations?.position!==void 0&&(this.reflowViewport.positions=[this.currentLocation.locations.position],this.lastLocationInView?.locations?.position!==void 0&&this.reflowViewport.positions.push(this.lastLocationInView.locations.position)))}async syncLocation(t){let e=t,i=this.findNearestPositions(e);this.currentLocation=i.first.copyWithLocations({progression:e.start}),this.lastLocationInView=i.last,this.updateViewport(e),this.notifyViewportChange(this.listeners.viewportChanged),this.listeners.positionChanged(this.currentLocation);let o=e.fragmentId?this.currentLocation.copyWithLocations({fragments:[`#${e.fragmentId}`]}):this.currentLocation;this._visibleFragmentIds=e.visibleFragmentIds??[],this._notifyTimelineChange(o),await this.framePool.update(this.pub,this.currentLocation,this.determineModules())}goBackward(t,e){if(this._isNavigating){e(!1);return}this._isNavigating=!0,this._layout===_.fixed?this.changeResource(-1).then(i=>{this._isNavigating=!1,e(i)}):this._cframes[0]?.msg?.send("go_prev",void 0,async i=>{if(i)this._isNavigating=!1,e(!0);else{let o=await this.changeResource(-1);this._isNavigating=!1,e(o)}})}goForward(t,e){if(this._isNavigating){e(!1);return}this._isNavigating=!0,this._layout===_.fixed?this.changeResource(1).then(i=>{this._isNavigating=!1,e(i)}):this._cframes[0]?.msg?.send("go_next",void 0,async i=>{if(i)this._isNavigating=!1,e(!0);else{let o=await this.changeResource(1);this._isNavigating=!1,e(o)}})}get currentLocator(){return this.currentLocation}get viewport(){return this._layout===_.fixed?this.framePool?this.framePool.viewport:{readingOrder:[],progressions:new Map,positions:null}:this.reflowViewport}get isScrollStart(){let t=this.viewport.readingOrder[0];return this.viewport.progressions.get(t)?.start===0}get isScrollEnd(){let t=this.viewport.readingOrder[this.viewport.readingOrder.length-1];return this.viewport.progressions.get(t)?.end===1}get canGoBackward(){let t=this.pub.readingOrder.items[0]?.href;return!(this.viewport.progressions.has(t)&&this.viewport.progressions.get(t)?.start===0)}get canGoForward(){let t=this.pub.readingOrder.items[this.pub.readingOrder.items.length-1]?.href;return!(this.viewport.progressions.has(t)&&this.viewport.progressions.get(t)?.end===1)}get readingProgression(){return this.currentProgression}async setLayout(t){this._layout!==t&&(this._layout=t,await this.framePool.update(this.pub,this.currentLocator,this.determineModules(),!0),this.attachListener())}get publication(){return this.pub}get timeline(){let t=this.pub.timeline;if(!this._timelineAugmented&&this.positions?.length){let e=this.positions,i=new Set;t.augment((o,n)=>{if(o.children?.length)return{};let s=n.href.indexOf("#"),a=s>=0?n.href.slice(0,s):n.href,h=s>=0?n.href.slice(s+1):void 0,l=e.filter(m=>m.href===a);if(!l.length)return{};let d=h?l.find(m=>m.locations.fragments?.[0]===h):void 0,c;return d?c=d:i.has(a)?c=void 0:(i.add(a),c=l.reduce((m,p)=>(p.locations.position??1/0)<(m.locations.position??1/0)?p:m)),c?{position:c.locations.position,scroll:d?.locations.progression}:{}}),this._timelineAugmented=!0}return this._wrappedTimeline||(this._wrappedTimeline=new Proxy(t,{get:(e,i)=>{if(i!=="navigableFrom"){let o=Reflect.get(e,i,e);return typeof o=="function"?o.bind(e):o}return o=>{let n=this._visibleFragmentIds,s=(n.length?e.locate(this.currentLocation.copyWithLocations({fragments:[`#${n[0]}`]})):void 0)??o,a=(n.length?e.locate(this.currentLocation.copyWithLocations({fragments:[`#${n[n.length-1]}`]})):void 0)??o;if(this.isScrollStart){let d=e.ancestors(s).find(c=>c.references.includes(this.currentLocation.href));d&&(s=d)}let h=e.navigableFrom(s).previous,l=e.navigableFrom(a).next;return{previous:h,next:l}}}})),this._wrappedTimeline}async loadLocator(t,e){let i=!1,o=_t(t.locations);if(t.text?.highlight?i=await new Promise((a,h)=>{this._cframes[0].msg.send("go_text",o?[t.text?.serialize(),o]:t.text?.serialize(),l=>a(l))}):o&&(i=await new Promise((a,h)=>{this._cframes[0].msg.send("go_text",["",o],l=>a(l))})),i){e(i);return}let n=_e(t.locations);if(n&&(i=await new Promise((a,h)=>{this._cframes[0].msg.send("go_id",n,l=>a(l))})),i){e(i);return}let s=t.locations?.progression;s&&s>0?i=await new Promise((a,h)=>{this._cframes[0].msg.send("go_progression",s,l=>a(l))}):i=!0,e(i)}completeLocator(t){if(!t.href){let e=!1;if(typeof t.locations.position=="number"){let i=this.positions.find(o=>o.locations.position===t.locations.position);i&&(t=i.copyWithLocations(t.locations),e=!0)}if(!e&&typeof t.locations?.totalProgression=="number"){let i=t.locations.totalProgression,o=0,n=1/0;for(let s=0;s<this.positions.length;s++){let a=this.positions[s].locations.totalProgression??s/this.positions.length,h=Math.abs(a-i);h<n&&(n=h,o=s)}t=this.positions[o].copyWithLocations(t.locations)}}return t}go(t,e,i){t=this.completeLocator(t);let o=t.href.split("#")[0],n=this.pub.readingOrder.findWithHref(o);if(!n)return i(this.listeners.handleLocator(t));if(this._isNavigating){i(!1);return}this._isNavigating=!0,this.currentLocation=this.positions.find(s=>s.href===n.href),this.apply().then(()=>this.loadLocator(t,s=>{this._isNavigating=!1,i(s)})).then(()=>{this.attachListener()}).catch(s=>{console.error(s),this._isNavigating=!1,i(!1)})}goLink(t,e,i){return this.go(t.locator,e,i)}_notifyTimelineChange(t){let e=this.timeline.locate(t),i=!nl(this._visibleFragmentIds,this._notifiedVisibleFragmentIds);(e!==this._currentTimelineItem||i)&&(this._currentTimelineItem=e,this._notifiedVisibleFragmentIds=this._visibleFragmentIds,this.listeners.timelineItemChanged(e))}};export{Ai as EpubNavigator,Me as HttpFetcher,q as Link,z as Locator,Ke as Manifest,qe as Publication};
