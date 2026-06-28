import PhotoSwipeLightbox from "./photoswipe/photoswipe-lightbox.esm.min.js";

// Global state
const state = {
  currentFolderHash: "",
  currentPage: 1,
  hasMore: false,
  isLoading: false,
  activeLightbox: null,
};

// IntersectionObserver for lazy loading images
const lazyImageObserver = new IntersectionObserver((entries, observer) => {
  entries.forEach((entry) => {
    if (entry.isIntersecting) {
      const img = entry.target;
      img.src = img.dataset.src;
      img.classList.add("loaded");
      observer.unobserve(img);
    }
  });
});

// Initialize folder tree click events
function initFolderTree() {
  const folderTree = document.querySelector(".folder-tree");
  if (!folderTree) return;

  folderTree.addEventListener("click", (e) => {
    // 1. Toggle button click detection
    const btn = e.target.closest(".toggle-btn");
    if (btn) {
      e.stopPropagation();
      const node = btn.closest(".tree-node");
      const children = node.querySelector(".tree-children");
      const chevron = btn.querySelector(".chevron");

      if (children) {
        children.classList.toggle("hidden");
        chevron.classList.toggle("open");
      }
      return;
    }

    // 2. Folder label click detection
    const label = e.target.closest(".tree-label");
    if (label) {
      const hash = label.dataset.hash;
      const hasImages = label.dataset.hasImages === "true";

      folderTree.querySelectorAll(".tree-label").forEach((l) =>
        l.classList.remove("active")
      );
      label.classList.add("active");

      document.querySelector(".sidebar").classList.remove("active");

      const folderNameEl = label.querySelector(".folder-name");
      const folderName = folderNameEl ? folderNameEl.textContent : "";
      selectFolder(hash, folderName, hasImages);
    }
  });
}

// Mobile menu toggle settings
function initMobileToggle() {
  const toggleBtn = document.getElementById("sidebar-toggle");
  const sidebar = document.querySelector(".sidebar");

  if (toggleBtn && sidebar) {
    toggleBtn.addEventListener("click", () => {
      sidebar.classList.toggle("active");
    });

    document.addEventListener("click", (e) => {
      if (
        window.matchMedia("(max-width: 768px)").matches &&
        !sidebar.contains(e.target) &&
        !toggleBtn.contains(e.target) &&
        sidebar.classList.contains("active")
      ) {
        sidebar.classList.remove("active");
      }
    });
  }
}

// PhotoSwipe initialization
function initPhotoSwipe() {
  if (state.activeLightbox) {
    state.activeLightbox.destroy();
  }

  const lightbox = new PhotoSwipeLightbox({
    gallery: "#gallery-grid",
    children: "a.image-card",
    pswpModule: () => import("./photoswipe/photoswipe.esm.min.js"),
    showHideAnimationType: "fade",
    imageClickAction: "zoom",
    tapAction: "zoom",
    doubleTapAction: "zoom",
    isClickableToZoom: () => true,
    initialZoomLevel: "fit",
    secondaryZoomLevel: (zoomLevelObject) => {
      if (!zoomLevelObject.itemData || !zoomLevelObject.itemData.w) {
        return 1.0;
      }
      const fitScale = zoomLevelObject.panAreaSize.x /
        zoomLevelObject.itemData.w;
      if (fitScale > 1) {
        return Math.min(4.0, fitScale);
      }
      return 1.0;
    },
    maxZoomLevel: (zoomLevelObject) => {
      if (!zoomLevelObject.itemData || !zoomLevelObject.itemData.w) {
        return 2.0;
      }
      const fitScale = zoomLevelObject.panAreaSize.x /
        zoomLevelObject.itemData.w;
      if (fitScale > 1) {
        return Math.max(6.0, fitScale);
      }
      return 2.5;
    },
  });

  // Custom caption element registration
  lightbox.on("uiRegister", () => {
    lightbox.pswp.ui.registerElement({
      name: "custom-caption",
      order: 9,
      isButton: false,
      appendTo: "root",
      onInit: (el, pswp) => {
        el.style.position = "absolute";
        el.style.left = "0";
        el.style.right = "0";
        el.style.bottom = "0";
        el.style.width = "100%";
        el.style.zIndex = "1000";
        el.style.pointerEvents = "none";

        pswp.on("change", () => {
          const currSlideElement = pswp.currSlide.data.element;
          el.replaceChildren();
          if (currSlideElement) {
            const caption = currSlideElement.querySelector(
              ".pswp-caption-content",
            );
            if (caption) {
              const captionContent = document.createElement("div");
              captionContent.className = "pswp__caption-content";
              Array.from(caption.childNodes).forEach((child) => {
                captionContent.appendChild(child.cloneNode(true));
              });
              el.appendChild(captionContent);
            }
          }
        });
      },
    });
  });

  lightbox.init();
  state.activeLightbox = lightbox;
}

// Directory selection handler
function selectFolder(hash, name, hasImages) {
  state.currentFolderHash = hash;
  state.currentPage = 1;
  state.hasMore = false;
  state.isLoading = false;

  document.getElementById("current-folder-title").textContent = name;
  const grid = document.getElementById("gallery-grid");
  grid.replaceChildren();

  const stats = document.getElementById("gallery-stats");
  stats.textContent = "";

  if (!hasImages) {
    const placeholder = document.createElement("div");
    placeholder.className = "placeholder-text";
    const p = document.createElement("p");
    p.textContent = "No images found in this directory directly.";
    placeholder.appendChild(p);
    grid.appendChild(placeholder);

    document.getElementById("sentinel").classList.add("hidden");
    return;
  }

  fetchImages();
}

// Fetch images from the server
async function fetchImages() {
  if (state.isLoading) return;
  state.isLoading = true;

  const sentinel = document.getElementById("sentinel");
  sentinel.classList.remove("hidden");

  try {
    const response = await fetch(
      `/api/images?hash=${state.currentFolderHash}&page=${state.currentPage}`,
    );
    if (!response.ok) {
      throw new Error(`Failed to fetch images: ${response.statusText}`);
    }

    const html = await response.text();
    const grid = document.getElementById("gallery-grid");

    if (state.currentPage === 1) {
      grid.replaceChildren();
    }

    state.hasMore = response.headers.get("X-Has-More") === "true";
    const totalCount = response.headers.get("X-Image-Count");
    if (totalCount) {
      document.getElementById("gallery-stats").textContent =
        `(${totalCount} images)`;
    }

    if (totalCount === "0" && state.currentPage === 1) {
      const placeholder = document.createElement("div");
      placeholder.className = "placeholder-text";
      const p = document.createElement("p");
      p.textContent = "No images available.";
      placeholder.appendChild(p);
      grid.appendChild(placeholder);
    } else {
      grid.insertAdjacentHTML("beforeend", html);
      grid.querySelectorAll(".lazy-img").forEach((img) => {
        if (!img.classList.contains("loaded")) {
          lazyImageObserver.observe(img);
        }
      });
    }

    if (state.hasMore) {
      state.currentPage++;
      sentinel.classList.remove("hidden");
    } else {
      sentinel.classList.add("hidden");
    }
  } catch (error) {
    console.error(error);
  } finally {
    state.isLoading = false;
    setTimeout(checkSentinel, 50);
  }
}

function checkSentinel() {
  if (state.isLoading || !state.hasMore) return;
  const sentinel = document.getElementById("sentinel");
  if (!sentinel || sentinel.classList.contains("hidden")) return;

  const rect = sentinel.getBoundingClientRect();
  const isVisible = rect.top <= window.innerHeight + 150;
  if (isVisible) {
    fetchImages();
  }
}

// Initialize infinite scroll
function initInfiniteScroll() {
  const sentinel = document.getElementById("sentinel");
  const scrollObserver = new IntersectionObserver((entries) => {
    const entry = entries[0];
    if (entry.isIntersecting && state.hasMore && !state.isLoading) {
      fetchImages();
    }
  }, {
    root: null,
    rootMargin: "150px",
    threshold: 0.1,
  });

  if (sentinel) {
    scrollObserver.observe(sentinel);
  }
}

// Theme toggle settings
function initTheme() {
  const toggleBtn = document.getElementById("theme-toggle");
  if (!toggleBtn) return;

  const savedTheme = localStorage.getItem("theme") || "dark";
  setTheme(savedTheme);

  toggleBtn.addEventListener("click", () => {
    const currentTheme =
      document.documentElement.getAttribute("data-theme") === "light"
        ? "light"
        : "dark";
    const nextTheme = currentTheme === "light" ? "dark" : "light";
    setTheme(nextTheme);
  });
}

function setTheme(theme) {
  if (theme === "light") {
    document.documentElement.setAttribute("data-theme", "light");
    localStorage.setItem("theme", "light");
  } else {
    document.documentElement.removeAttribute("data-theme");
    localStorage.setItem("theme", "dark");
  }
}

// Grid size slider settings
function initGridSize() {
  const slider = document.getElementById("grid-size-slider");
  if (!slider) return;

  const savedSize = localStorage.getItem("grid-size") || "220";
  slider.value = savedSize;
  document.documentElement.style.setProperty("--grid-size", `${savedSize}px`);

  slider.addEventListener("input", (e) => {
    document.documentElement.style.setProperty(
      "--grid-size",
      `${e.target.value}px`,
    );
  });

  slider.addEventListener("change", (e) => {
    localStorage.setItem("grid-size", e.target.value);
  });
}

// Initialize all components when the DOM is fully loaded
document.addEventListener("DOMContentLoaded", () => {
  initTheme();
  initGridSize();
  initFolderTree();
  initMobileToggle();
  initInfiniteScroll();
  initPhotoSwipe();

  const rootFolder = document.querySelector(
    ".folder-tree > .tree-node > .tree-row-wrapper > .tree-label",
  );
  if (rootFolder) {
    rootFolder.click();
  } else {
    const fallbackFolder = document.querySelector(".tree-label");
    if (fallbackFolder) fallbackFolder.click();
  }
});
