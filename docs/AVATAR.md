# Avatar module

An optional Live2D character beside the Hermes chat. It blinks, sways, shows a mood from what she says, and moves her
mouth with her voice. Everything is drawn in the browser; the server only serves the files, to administrators.

## Set up

1. Switch on **Avatar** in Settings, Modules (it needs the Hermes module).
2. Put one folder per model in the `avatar` folder next to `compose.yaml` (mounted at `/avatars`). A model is any folder
   that holds a `.model3.json` with its `.moc3`, textures and optional `.exp3.json` expression files. Folders can be
   nested up to four levels deep.
3. Put the Live2D Cubism core at `avatar/_runtime/live2dcubismcore.min.js`. It has its own licence and is not part of
   Mochil: download it from <https://cubism.live2d.com/sdk-web/cubismcore/live2dcubismcore.min.js> and read
   <https://www.live2d.com/eula/live2d-proprietary-software-license-agreement_en.html>.
4. Choose the model in Settings, Avatar, then turn on **Show avatar** in the chat's voice options.

To use another folder, set `"avatar": {"directory": "/some/path"}` in the config.

## Notes

- The `avatar` folder is ignored by git: models are bought or licensed assets.
- The model needs the standard parameter `ParamMouthOpenY` to move its mouth. Blinking, head sway and eye movement use
  `ParamEyeLOpen`, `ParamEyeROpen`, `ParamAngleX/Y/Z` and `ParamEyeBallX` when they exist.
- Expressions are picked by name (English, or the usual Chinese names): blush, love, angry, sad (tears), surprise,
  question. A model without a matching file simply keeps its face.
- The chat page allows WebAssembly (`'wasm-unsafe-eval'`) because the Cubism core needs it. Nothing else is relaxed.

## Background and position

In the Hermes chat settings (the gear), with an avatar showing:

- **Background**: Soft (default), Cosy room, Night window, Sunset, Studio, Mint, or **My picture**. The scenes are plain
  gradients drawn by the page. My picture is one JPEG, PNG or WebP you upload (shrunk in the browser first, 6 MB at most);
  it is stored next to the avatar state file (`avatar-background`), not in the avatar folder, which stays read only.
- **Position**: size, left/right and up/down, kept per avatar. Reset position puts it back.

Saved on the server, so the choice follows you between devices that show the avatar. Administrators only, like the rest of
the avatar API (`PUT /api/avatar/look`, `GET|PUT|DELETE /api/avatar/background`).
