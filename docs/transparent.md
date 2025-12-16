## Transparent Window for wails

### Windows Example
```go

BackgroundColour: nil,
WindowStartState: options.Normal,
Frameless:        true, // 无边框窗口
Windows: &windows.Options{
    WebviewIsTransparent: true, // WebView 透明
    WindowIsTranslucent: true, // 窗口半透明，建議false
    DisableFramelessWindowDecorations: true, // 禁用窗口装饰
},
```

### macOS Example
```go
Mac: &mac.Options{
    WebviewIsTransparent: true,
    WindowIsTranslucent:  true, // 有時會不透明，建議false
},
```

Appearance type
You can specify the application's appearance.

Value	Description
DefaultAppearance	DefaultAppearance uses the default system value
NSAppearanceNameAqua	The standard light system appearance
NSAppearanceNameDarkAqua	The standard dark system appearance
NSAppearanceNameVibrantLight	The light vibrant appearance
NSAppearanceNameAccessibilityHighContrastAqua	A high-contrast version of the standard light system appearance
NSAppearanceNameAccessibilityHighContrastDarkAqua	A high-contrast version of the standard dark system appearance
NSAppearanceNameAccessibilityHighContrastVibrantLight	A high-contrast version of the light vibrant appearance
NSAppearanceNameAccessibilityHighContrastVibrantDark	A high-contrast version of the dark vibrant appearance
Example:

Mac: &mac.Options{
    Appearance: mac.NSAppearanceNameDarkAqua,
}

WebviewIsTransparent
Setting this to true will make the webview background transparent when an alpha value of 0 is used. This means that if you use rgba(0,0,0,0) for background-color in your CSS, the host window will show through. Often combined with WindowIsTranslucent to make frosty-looking applications.

Name: WebviewIsTransparent
Type: bool

WindowIsTranslucent
Setting this to true will make the window background translucent. Often combined with WebviewIsTransparent to make frosty-looking applications.

Name: WindowIsTranslucent
Type: bool