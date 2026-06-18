/**
 * Theme manager — shared by index and detail pages.
 */
(function (global) {
    "use strict";

    const STORAGE_KEY = "phantom-theme";
    const DEFAULT_THEME = "obsidian";

    const THEMES = [
        { id: "obsidian", label: "Obsidian", swatch: "#6366f1" },
        { id: "light", label: "Light", swatch: "#f8fafc" },
        { id: "midnight", label: "Midnight", swatch: "#38bdf8" },
        { id: "nord", label: "Nord", swatch: "#88c0d0" },
        { id: "emerald", label: "Emerald", swatch: "#34d399" },
        { id: "sunset", label: "Sunset", swatch: "#fb7185" },
    ];

    function getTheme() {
        return localStorage.getItem(STORAGE_KEY) || DEFAULT_THEME;
    }

    function applyTheme(themeId) {
        const valid = THEMES.some(t => t.id === themeId);
        const theme = valid ? themeId : DEFAULT_THEME;
        document.documentElement.setAttribute("data-theme", theme);
        localStorage.setItem(STORAGE_KEY, theme);
        document.querySelectorAll("[data-theme-select]").forEach(el => {
            el.value = theme;
        });
        return theme;
    }

    function initTheme() {
        applyTheme(getTheme());
        window.addEventListener("storage", (ev) => {
            if (ev.key === STORAGE_KEY && ev.newValue) {
                applyTheme(ev.newValue);
            }
        });
    }

    function bindThemeSelect(selectEl) {
        if (!selectEl) return;
        selectEl.innerHTML = THEMES.map(t =>
            `<option value="${t.id}">${t.label}</option>`).join("");
        selectEl.value = getTheme();
        selectEl.addEventListener("change", () => applyTheme(selectEl.value));
    }

    global.PhantomTheme = { THEMES, getTheme, applyTheme, initTheme, bindThemeSelect };
})(window);
