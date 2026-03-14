function createMacomaApp() {
  return {
    file: null,
    previewUrl: "",
    livePreview: false,
    busy: false,
    error: "",
    status: "",
    previewAbortController: null,
    debounceTimer: null,
    form: {
      delimiter_strategy: "color",
      border_delimiter_color: "#000000",
      border_delimiter_tolerance: "10",
      color_delimiter_tolerance: "10",
      max_colors: "10"
    },

    onFileChange(event) {
      this.file = event.target.files && event.target.files[0] ? event.target.files[0] : null;
      this.error = "";
      if (!this.file) {
        this.previewUrl = "";
        this.status = "No image selected.";
        return;
      }
      this.status = `Selected: ${this.file.name}`;
      this.onSettingsChange();
    },

    onSettingsChange() {
      if (!this.livePreview || !this.file) {
        return;
      }
      clearTimeout(this.debounceTimer);
      this.debounceTimer = setTimeout(() => this.requestPreview(), 350);
    },

    buildFormData() {
      const fd = new FormData();
      fd.append("image", this.file);
      fd.append("delimiter_strategy", this.form.delimiter_strategy);
      fd.append("border_delimiter_color", this.form.border_delimiter_color);
      fd.append("border_delimiter_tolerance", String(this.form.border_delimiter_tolerance));
      fd.append("color_delimiter_tolerance", String(this.form.color_delimiter_tolerance));
      fd.append("max_colors", String(this.form.max_colors));
      return fd;
    },

    async requestPreview() {
      if (!this.file) {
        this.error = "Please select an input image.";
        return;
      }
      if (this.previewAbortController) {
        this.previewAbortController.abort();
      }
      this.previewAbortController = new AbortController();
      this.error = "";
      this.status = "Generating preview...";

      try {
        const resp = await fetch("/api/preview", {
          method: "POST",
          body: this.buildFormData(),
          signal: this.previewAbortController.signal
        });
        if (!resp.ok) {
          throw await this.toError(resp);
        }
        const blob = await resp.blob();
        if (this.previewUrl) {
          URL.revokeObjectURL(this.previewUrl);
        }
        this.previewUrl = URL.createObjectURL(blob);
        this.status = "Preview updated.";
      } catch (err) {
        if (err.name === "AbortError") {
          return;
        }
        this.error = err.message || "Preview request failed.";
        this.status = "";
      }
    },

    async renderFinal() {
      if (!this.file) {
        this.error = "Please select an input image.";
        return;
      }
      this.error = "";
      this.status = "Rendering full-quality output...";
      this.busy = true;

      try {
        const resp = await fetch("/api/render", {
          method: "POST",
          body: this.buildFormData()
        });
        if (!resp.ok) {
          throw await this.toError(resp);
        }
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = "macoma-output.png";
        a.click();
        URL.revokeObjectURL(url);
        this.status = "Render complete. Download started.";
      } catch (err) {
        this.error = err.message || "Render request failed.";
        this.status = "";
      } finally {
        this.busy = false;
      }
    },

    async toError(resp) {
      try {
        const data = await resp.json();
        if (data && data.error) {
          return new Error(data.error);
        }
      } catch (_) {
      }
      return new Error(`Request failed (${resp.status})`);
    }
  };
}

document.addEventListener("alpine:init", () => {
  Alpine.data("macomaApp", createMacomaApp);
});
