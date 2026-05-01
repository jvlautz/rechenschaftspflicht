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
  let p = 100; // Start with high initial uncertainty to show convergence
  
  return sorted.map((d, i) => {
    // 1. Sliding Window Parameter Estimation
    const start = Math.max(0, i - windowSize);
    const window = sorted.slice(start, i + 1).map(v => v.valueNum);
    
    // R: Measurement Noise (local variance)
    const R = getVariance(window) || 1.0;
    
    // Q: Process Noise (variance of consecutive steps)
    const steps = [];
    for (let j = 1; j < window.length; j++) steps.push(window[j] - window[j-1]);
    const Q = getVariance(steps) || 0.1;

    // 2. Predict
    p = p + Q;
    
    // 3. Update (Correct)
    const k = p / (p + R);
    x = x + k * (d.valueNum - x);
    p = (1 - k) * p;
    
    return { 
      ...d, 
      kalmanValue: x, 
      error: Math.sqrt(p) 
    };
  });
}




// --- Application State ---
let allEvents = [];
let activeUsers = new Set();

async function init() {
  // Replace with your actual data endpoint
  const response = await fetch("/events.json");
  if (!response.ok) throw new Error("Failed to load data");
  
  allEvents = await response.json();
  
  const users = [...new Set(allEvents.map(d => d.recordedBy))];
  activeUsers = new Set(users); // Default all users to "on"

  renderControls(users);
  updatePlot();
}

// --- UI: Toggle Controls ---
function renderControls(users) {
  let container = document.querySelector("#controls");
  if (!container) {
    container = document.createElement("div");
    container.id = "controls";
    container.style.cssText = "margin-bottom: 20px; padding: 10px; background: #f4f4f4; border-radius: 8px; font-family: sans-serif;";
    const plotDiv = document.querySelector("#myplot");
    plotDiv.parentNode.insertBefore(container, plotDiv);
  }

  container.innerHTML = "<strong>Toggle Users: </strong>";

  users.forEach(user => {
    const label = document.createElement("label");
    label.style.cssText = "margin-right: 15px; cursor: pointer; user-select: none;";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = true;
    checkbox.style.marginRight = "5px";
    checkbox.onchange = () => {
      if (checkbox.checked) activeUsers.add(user);
      else activeUsers.delete(user);
      updatePlot();
    };

    label.append(checkbox, user);
    container.append(label);
  });
}

// --- Rendering: Update Plot ---
function updatePlot() {
  // Filter events by active users
  const filteredEvents = allEvents.filter(e => activeUsers.has(e.recordedBy));
  const tags = [...new Set(filteredEvents.map(d => d.tag))];
  
  // Process data with the dynamic Kalman filter per user/tag group
  const processedData = [];
  for (const tag of tags) {
    activeUsers.forEach(user => {
      const group = filteredEvents.filter(e => e.tag === tag && e.recordedBy === user);
      if (group.length > 0) {
        processedData.push(...applyDynamicKalmanFilter(group));
      }
    });
  }

  const plot = Plot.plot({
    facet: { data: processedData, y: "tag" },
    y: { 
      label: "Estimated Value", 
      facet: "separate", 
      reserve: 40 // This tells Plot each facet gets its own scale
    } ,
    color: { legend: true },
    marks: [
      // Dynamic Confidence Band (2*sigma)
      Plot.areaY(processedData, {
        x: (d) => new Date(d.recordedAt),
        y1: (d) => d.kalmanValue - 2 * d.error,
        y2: (d) => d.kalmanValue + 2 * d.error,
        fill: "recordedBy",
        fillOpacity: 0.12,
      }),
      // Kalman Estimated Line
      Plot.line(processedData, {
        x: (d) => new Date(d.recordedAt),
        y: "kalmanValue",
        stroke: "recordedBy",
        strokeWidth: 2,
      }),
      // Raw Measurement Points
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
  div.innerHTML = ""; // Clear existing plot
  div.append(plot);
}

init().catch(console.error);
