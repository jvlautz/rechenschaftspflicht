import * as Plot from "https://cdn.jsdelivr.net/npm/@observablehq/plot@0.6/+esm";

// --- Helper: Statistical Variance ---
const getVariance = (arr) => {
  if (arr.length < 2) return 0;
  const mean = arr.reduce((a, b) => a + b, 0) / arr.length;
  return arr.reduce((a, b) => a + Math.pow(b - mean, 2), 0) / arr.length;
};

// --- Core: Dynamic Sliding Window Kalman Filter ---
function applyDynamicKalmanFilter(data, windowSize = 12) {
  if (data.length === 0) return [];
  const sorted = [...data].sort((a, b) => new Date(a.recordedAt) - new Date(b.recordedAt));
  let x = sorted[0].valueNum; 
  let p = 100; 
  
  return sorted.map((d, i) => {
    const start = Math.max(0, i - windowSize);
    const window = sorted.slice(start, i + 1).map(v => v.valueNum);
    const R = getVariance(window) || 1.0;
    const steps = [];
    for (let j = 1; j < window.length; j++) steps.push(window[j] - window[j-1]);
    const Q = getVariance(steps) || 0.1;
    p = p + Q;
    const k = p / (p + R);
    x = x + k * (d.valueNum - x);
    p = (1 - k) * p;
    return { ...d, kalmanValue: x, error: Math.sqrt(p) };
  });
}

// --- Application State ---
let allEvents = [];
let activeUsers = new Set();
let activeTags = new Set(); // Track which tags are visible

async function init() {
  const response = await fetch("/events.json");
  if (!response.ok) throw new Error("Failed to load data");
  
  allEvents = await response.json();
  
  const users = [...new Set(allEvents.map(d => d.recordedBy))];
  const tags = [...new Set(allEvents.map(d => d.tag))];
  
  activeUsers = new Set(users);
  activeTags = new Set(tags); // Default all tags to "on"

  renderControls(users, tags);
  updatePlot();
}

// --- UI: Toggle Controls ---
function renderControls(users, tags) {
  let container = document.querySelector("#controls");
  if (!container) {
    container = document.createElement("div");
    container.id = "controls";
    container.style.cssText = "margin-bottom: 20px; padding: 15px; background: #f9f9f9; border-radius: 8px; font-family: sans-serif; border: 1px solid #ddd;";
    const plotDiv = document.querySelector("#myplot");
    plotDiv.parentNode.insertBefore(container, plotDiv);
  }

  container.innerHTML = ""; // Clear for re-render

  // Helper to create checkbox groups
  const createSection = (title, items, activeSet) => {
    const section = document.createElement("div");
    section.style.marginBottom = "10px";
    section.innerHTML = `<strong>${title}: </strong>`;
    items.forEach(item => {
      const label = document.createElement("label");
      label.style.cssText = "margin-right: 15px; cursor: pointer; font-size: 14px;";
      const cb = document.createElement("input");
      cb.type = "checkbox";
      cb.checked = activeSet.has(item);
      cb.style.marginRight = "5px";
      cb.onchange = () => {
        if (cb.checked) activeSet.add(item);
        else activeSet.delete(item);
        updatePlot();
      };
      label.append(cb, item);
      section.append(label);
    });
    return section;
  };

  container.append(createSection("Filter Users", users, activeUsers));
  container.append(createSection("Filter Tags (Subplots)", tags, activeTags));
}

// --- Rendering: Update Plot ---
function updatePlot() {
  // 1. Filter by both active Users AND active Tags
  const filteredEvents = allEvents.filter(e => 
    activeUsers.has(e.recordedBy) && activeTags.has(e.tag)
  );
  
  // 2. Process data only for visible tags/users
  const processedData = [];
  activeTags.forEach(tag => {
    activeUsers.forEach(user => {
      const group = filteredEvents.filter(e => e.tag === tag && e.recordedBy === user);
      if (group.length > 0) {
        processedData.push(...applyDynamicKalmanFilter(group));
      }
    });
  });

  const plot = Plot.plot({
    facet: { data: processedData, y: "tag" },
    y: { 
      label: "Estimated Value", 
      facet: "separate", 
      reserve: 40 // This tells Plot each facet gets its own scale
    } ,
    color: { legend: true },
    marks: [
      // Confidence Band
      Plot.areaY(processedData, {
        x: (d) => new Date(d.recordedAt),
        y1: (d) => d.kalmanValue - 2 * d.error,
        y2: (d) => d.kalmanValue + 2 * d.error,
        fill: "recordedBy",
        fillOpacity: 0.1,
      }),
      // Kalman Line
      Plot.line(processedData, {
        x: (d) => new Date(d.recordedAt),
        y: "kalmanValue",
        stroke: "recordedBy",
        strokeWidth: 2,
      }),
      // Raw Dots
      Plot.dot(processedData, {
        x: (d) => new Date(d.recordedAt),
        y: "valueNum",
        stroke: "recordedBy",
        r: 3,
        strokeOpacity: 0.8
      }),
      Plot.gridX(),
      Plot.gridY(),
    ],
    x: { label: "Recorded Time →" },
    y: { label: "Estimated Value", facet: { share: false } },
    marginLeft: 80,
    marginBottom: 50,
    height: 800,
    width: 1000,
  });

  const div = document.querySelector("#myplot");
  div.innerHTML = ""; 
  div.append(plot);
}

init().catch(console.error);
