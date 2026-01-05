// Package render provides HTML rendering functionality for network graphs.
package render

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/ddl-r-abdulaziz/dnmap/pkg/graph"
)

// HTMLRenderer renders network graphs to interactive HTML pages.
type HTMLRenderer struct{}

// NewHTMLRenderer creates a new HTML renderer.
func NewHTMLRenderer() *HTMLRenderer {
	return &HTMLRenderer{}
}

// Render converts a NetworkGraph to an interactive HTML page.
func (r *HTMLRenderer) Render(g *graph.NetworkGraph) (string, error) {
	graphJSON, err := json.Marshal(g)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("graph").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"GraphData": string(graphJSON),
	}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Domino Network Map</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&family=Outfit:wght@300;400;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #0a0e14;
            --bg-secondary: #121820;
            --bg-tertiary: #1a222d;
            --accent-cyan: #39bae6;
            --accent-orange: #ff8f40;
            --accent-green: #7fd962;
            --accent-purple: #c792ea;
            --accent-red: #f07178;
            --accent-yellow: #ffcc66;
            --text-primary: #e6e6e6;
            --text-secondary: #626a73;
            --border-color: #2a3444;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Outfit', sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            overflow: hidden;
            height: 100vh;
        }
        
        .header {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            height: 56px;
            background: linear-gradient(180deg, var(--bg-secondary) 0%, rgba(18, 24, 32, 0.95) 100%);
            backdrop-filter: blur(12px);
            border-bottom: 1px solid var(--border-color);
            display: flex;
            align-items: center;
            padding: 0 24px;
            z-index: 100;
            gap: 24px;
        }
        
        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        
        .logo-icon {
            width: 32px;
            height: 32px;
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .logo-icon svg {
            width: 20px;
            height: 20px;
        }
        
        .logo-text {
            font-weight: 700;
            font-size: 18px;
            letter-spacing: -0.02em;
        }
        
        .stats {
            display: flex;
            gap: 16px;
            margin-left: auto;
        }
        
        .stat {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 6px 12px;
            background: var(--bg-tertiary);
            border-radius: 6px;
            font-size: 13px;
        }
        
        .stat-value {
            font-family: 'JetBrains Mono', monospace;
            font-weight: 600;
            color: var(--accent-cyan);
        }
        
        .stat-label {
            color: var(--text-secondary);
        }
        
        .controls {
            display: flex;
            gap: 8px;
        }
        
        .btn {
            padding: 8px 16px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            color: var(--text-primary);
            font-family: 'Outfit', sans-serif;
            font-size: 13px;
            cursor: pointer;
            transition: all 0.15s ease;
        }
        
        .btn:hover {
            background: var(--border-color);
            border-color: var(--accent-cyan);
        }
        
        .btn-primary {
            background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
            border: none;
            font-weight: 600;
        }
        
        .btn-primary:hover {
            opacity: 0.9;
            transform: translateY(-1px);
        }
        
        #canvas-container {
            position: fixed;
            top: 56px;
            left: 0;
            right: 0;
            bottom: 0;
            background: 
                radial-gradient(circle at 20% 30%, rgba(57, 186, 230, 0.03) 0%, transparent 50%),
                radial-gradient(circle at 80% 70%, rgba(199, 146, 234, 0.03) 0%, transparent 50%),
                var(--bg-primary);
        }
        
        #canvas {
            width: 100%;
            height: 100%;
        }
        
        .tooltip {
            position: fixed;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 12px 16px;
            font-size: 13px;
            max-width: 400px;
            pointer-events: none;
            opacity: 0;
            transition: opacity 0.15s ease;
            z-index: 200;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
        }
        
        .tooltip.visible {
            opacity: 1;
        }
        
        .tooltip-title {
            font-weight: 600;
            font-size: 14px;
            margin-bottom: 8px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .tooltip-badge {
            font-size: 11px;
            padding: 2px 8px;
            border-radius: 4px;
            font-family: 'JetBrains Mono', monospace;
            font-weight: 600;
        }
        
        .badge-deployment { background: rgba(127, 217, 98, 0.2); color: var(--accent-green); }
        .badge-statefulset { background: rgba(199, 146, 234, 0.2); color: var(--accent-purple); }
        .badge-daemonset { background: rgba(255, 143, 64, 0.2); color: var(--accent-orange); }
        .badge-port { background: rgba(57, 186, 230, 0.2); color: var(--accent-cyan); }
        
        .tooltip-row {
            display: flex;
            gap: 8px;
            margin-top: 6px;
        }
        
        .tooltip-label {
            color: var(--text-secondary);
            min-width: 70px;
        }
        
        .tooltip-value {
            font-family: 'JetBrains Mono', monospace;
            color: var(--text-primary);
        }
        
        .tooltip-rule {
            margin-top: 10px;
            padding-top: 10px;
            border-top: 1px solid var(--border-color);
            font-family: 'JetBrains Mono', monospace;
            font-size: 11px;
            color: var(--accent-yellow);
            line-height: 1.5;
        }
        
        .legend {
            position: fixed;
            bottom: 24px;
            left: 24px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 16px 20px;
            z-index: 100;
        }
        
        .legend-title {
            font-size: 12px;
            color: var(--text-secondary);
            margin-bottom: 12px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        
        .legend-items {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }
        
        .legend-item {
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 13px;
        }
        
        .legend-color {
            width: 12px;
            height: 12px;
            border-radius: 3px;
        }
        
        .minimap {
            position: fixed;
            bottom: 24px;
            right: 24px;
            width: 180px;
            height: 120px;
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            overflow: hidden;
            z-index: 100;
        }
        
        #minimap-canvas {
            width: 100%;
            height: 100%;
        }
        
        .search-container {
            position: relative;
        }
        
        .search-input {
            padding: 8px 12px 8px 36px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            color: var(--text-primary);
            font-family: 'Outfit', sans-serif;
            font-size: 13px;
            width: 220px;
            outline: none;
            transition: all 0.15s ease;
        }
        
        .search-input:focus {
            border-color: var(--accent-cyan);
            background: var(--bg-secondary);
        }
        
        .search-input::placeholder {
            color: var(--text-secondary);
        }
        
        .search-icon {
            position: absolute;
            left: 12px;
            top: 50%;
            transform: translateY(-50%);
            color: var(--text-secondary);
        }
    </style>
</head>
<body>
    <header class="header">
        <div class="logo">
            <div class="logo-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2">
                    <circle cx="12" cy="12" r="3"/>
                    <path d="M12 2v4m0 12v4M2 12h4m12 0h4"/>
                    <path d="M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
                </svg>
            </div>
            <span class="logo-text">dnmap</span>
        </div>
        
        <div class="search-container">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="M21 21l-4.35-4.35"/>
            </svg>
            <input type="text" class="search-input" placeholder="Search workloads..." id="search-input">
        </div>
        
        <div class="stats">
            <div class="stat">
                <span class="stat-value" id="node-count">0</span>
                <span class="stat-label">workloads</span>
            </div>
            <div class="stat">
                <span class="stat-value" id="edge-count">0</span>
                <span class="stat-label">connections</span>
            </div>
        </div>
        
        <div class="controls">
            <button class="btn" onclick="resetView()">Reset View</button>
            <button class="btn" onclick="togglePhysics()">Toggle Physics</button>
            <button class="btn btn-primary" onclick="exportPNG()">Export PNG</button>
        </div>
    </header>
    
    <div id="canvas-container">
        <canvas id="canvas"></canvas>
    </div>
    
    <div class="tooltip" id="tooltip"></div>
    
    <div class="legend">
        <div class="legend-title">Workload Types</div>
        <div class="legend-items">
            <div class="legend-item">
                <div class="legend-color" style="background: #7fd962;"></div>
                <span>Deployment</span>
            </div>
            <div class="legend-item">
                <div class="legend-color" style="background: #c792ea;"></div>
                <span>StatefulSet</span>
            </div>
            <div class="legend-item">
                <div class="legend-color" style="background: #ff8f40;"></div>
                <span>DaemonSet</span>
            </div>
            <div class="legend-item">
                <div class="legend-color" style="background: #39bae6;"></div>
                <span>Port</span>
            </div>
        </div>
    </div>
    
    <div class="minimap">
        <canvas id="minimap-canvas"></canvas>
    </div>
    
    <script>
    const graphData = {{.GraphData}};
    
    // Canvas setup
    const canvas = document.getElementById('canvas');
    const ctx = canvas.getContext('2d');
    const minimapCanvas = document.getElementById('minimap-canvas');
    const minimapCtx = minimapCanvas.getContext('2d');
    const tooltip = document.getElementById('tooltip');
    
    let width, height;
    let dpr = window.devicePixelRatio || 1;
    
    // View state
    let panX = 0, panY = 0;
    let zoom = 1;
    let isDragging = false;
    let isPanning = false;
    let dragNode = null;
    let dragOffsetX = 0, dragOffsetY = 0;
    let lastMouseX = 0, lastMouseY = 0;
    let physicsEnabled = true;
    
    // Colors
    const colors = {
        Deployment: '#7fd962',
        StatefulSet: '#c792ea',
        DaemonSet: '#ff8f40',
        Pod: '#f07178',
        port: '#39bae6',
        edge: 'rgba(57, 186, 230, 0.4)',
        edgeHover: 'rgba(57, 186, 230, 0.8)',
    };
    
    // Node physics
    class GraphNode {
        constructor(data) {
            this.data = data;
            this.x = Math.random() * 800 - 400;
            this.y = Math.random() * 600 - 300;
            this.vx = 0;
            this.vy = 0;
            this.radius = data.type === 'port' ? 8 : 24;
            this.fixed = false;
        }
    }
    
    // Initialize nodes
    const nodes = new Map();
    const workloadNodes = [];
    const portNodes = [];
    
    graphData.nodes.forEach(n => {
        const node = new GraphNode(n);
        nodes.set(n.id, node);
        if (n.type === 'workload') {
            workloadNodes.push(node);
        } else {
            portNodes.push(node);
        }
    });
    
    // Position port nodes relative to their parents
    portNodes.forEach(portNode => {
        const parent = nodes.get(portNode.data.parent);
        if (parent) {
            portNode.x = parent.x + (Math.random() - 0.5) * 60;
            portNode.y = parent.y + (Math.random() - 0.5) * 60;
        }
    });
    
    // Edges
    const edges = graphData.edges.map(e => ({
        ...e,
        sourceNode: nodes.get(e.source),
        targetNode: nodes.get(e.target)
    })).filter(e => e.sourceNode && e.targetNode);
    
    // Update stats
    document.getElementById('node-count').textContent = workloadNodes.length;
    document.getElementById('edge-count').textContent = edges.length;
    
    function resize() {
        const rect = canvas.parentElement.getBoundingClientRect();
        width = rect.width;
        height = rect.height;
        
        canvas.width = width * dpr;
        canvas.height = height * dpr;
        canvas.style.width = width + 'px';
        canvas.style.height = height + 'px';
        ctx.scale(dpr, dpr);
        
        minimapCanvas.width = 180 * dpr;
        minimapCanvas.height = 120 * dpr;
        minimapCtx.scale(dpr, dpr);
        
        // Center view if first resize
        if (panX === 0 && panY === 0) {
            panX = width / 2;
            panY = height / 2;
        }
    }
    
    function worldToScreen(x, y) {
        return {
            x: x * zoom + panX,
            y: y * zoom + panY
        };
    }
    
    function screenToWorld(x, y) {
        return {
            x: (x - panX) / zoom,
            y: (y - panY) / zoom
        };
    }
    
    function applyPhysics() {
        if (!physicsEnabled) return;
        
        const damping = 0.85;
        const repulsion = 5000;
        const attraction = 0.01;
        const centerForce = 0.001;
        
        // Repulsion between workload nodes
        for (let i = 0; i < workloadNodes.length; i++) {
            for (let j = i + 1; j < workloadNodes.length; j++) {
                const a = workloadNodes[i];
                const b = workloadNodes[j];
                
                let dx = b.x - a.x;
                let dy = b.y - a.y;
                let dist = Math.sqrt(dx * dx + dy * dy) || 1;
                let force = repulsion / (dist * dist);
                
                let fx = (dx / dist) * force;
                let fy = (dy / dist) * force;
                
                if (!a.fixed) { a.vx -= fx; a.vy -= fy; }
                if (!b.fixed) { b.vx += fx; b.vy += fy; }
            }
        }
        
        // Edge attraction
        edges.forEach(edge => {
            const source = edge.sourceNode;
            const target = edge.targetNode;
            const parent = nodes.get(target.data.parent);
            
            if (source && parent) {
                let dx = parent.x - source.x;
                let dy = parent.y - source.y;
                let dist = Math.sqrt(dx * dx + dy * dy);
                let force = dist * attraction;
                
                if (!source.fixed) { source.vx += dx * force; source.vy += dy * force; }
                if (!parent.fixed) { parent.vx -= dx * force; parent.vy -= dy * force; }
            }
        });
        
        // Center force
        workloadNodes.forEach(node => {
            if (!node.fixed) {
                node.vx -= node.x * centerForce;
                node.vy -= node.y * centerForce;
            }
        });
        
        // Apply velocity
        workloadNodes.forEach(node => {
            if (!node.fixed) {
                node.vx *= damping;
                node.vy *= damping;
                node.x += node.vx;
                node.y += node.vy;
            }
        });
        
        // Port nodes follow parents
        portNodes.forEach(portNode => {
            const parent = nodes.get(portNode.data.parent);
            if (parent) {
                const angle = getPortAngle(portNode, parent);
                const dist = 40;
                portNode.x = parent.x + Math.cos(angle) * dist;
                portNode.y = parent.y + Math.sin(angle) * dist;
            }
        });
    }
    
    function getPortAngle(portNode, parent) {
        const siblings = portNodes.filter(p => p.data.parent === parent.data.id);
        const idx = siblings.indexOf(portNode);
        const total = siblings.length;
        const startAngle = -Math.PI / 2;
        const spread = Math.PI * 2 / Math.max(total, 1);
        return startAngle + idx * spread;
    }
    
    let hoveredNode = null;
    let hoveredEdge = null;
    let searchTerm = '';
    
    function draw() {
        ctx.clearRect(0, 0, width, height);
        
        // Draw grid
        ctx.strokeStyle = 'rgba(42, 52, 68, 0.3)';
        ctx.lineWidth = 1;
        const gridSize = 50 * zoom;
        const offsetX = panX % gridSize;
        const offsetY = panY % gridSize;
        
        ctx.beginPath();
        for (let x = offsetX; x < width; x += gridSize) {
            ctx.moveTo(x, 0);
            ctx.lineTo(x, height);
        }
        for (let y = offsetY; y < height; y += gridSize) {
            ctx.moveTo(0, y);
            ctx.lineTo(width, y);
        }
        ctx.stroke();
        
        // Draw edges
        edges.forEach(edge => {
            const source = edge.sourceNode;
            const target = edge.targetNode;
            
            const start = worldToScreen(source.x, source.y);
            const end = worldToScreen(target.x, target.y);
            
            const isHovered = hoveredEdge === edge;
            const opacity = isHovered ? 1 : 0.4;
            
            // Draw curved line
            ctx.beginPath();
            const midX = (start.x + end.x) / 2;
            const midY = (start.y + end.y) / 2;
            const dx = end.x - start.x;
            const dy = end.y - start.y;
            const len = Math.sqrt(dx * dx + dy * dy);
            const offset = len * 0.15;
            const ctrlX = midX - dy / len * offset;
            const ctrlY = midY + dx / len * offset;
            
            ctx.moveTo(start.x, start.y);
            ctx.quadraticCurveTo(ctrlX, ctrlY, end.x, end.y);
            ctx.strokeStyle = isHovered ? 'rgba(57, 186, 230, 0.9)' : 'rgba(57, 186, 230, ' + opacity + ')';
            ctx.lineWidth = isHovered ? 3 : 1.5;
            ctx.stroke();
            
            // Draw arrow
            const arrowSize = 8;
            const t = 0.85;
            const ax = (1-t)*(1-t)*start.x + 2*(1-t)*t*ctrlX + t*t*end.x;
            const ay = (1-t)*(1-t)*start.y + 2*(1-t)*t*ctrlY + t*t*end.y;
            const tangentX = 2*(1-t)*(ctrlX-start.x) + 2*t*(end.x-ctrlX);
            const tangentY = 2*(1-t)*(ctrlY-start.y) + 2*t*(end.y-ctrlY);
            const angle = Math.atan2(tangentY, tangentX);
            
            ctx.beginPath();
            ctx.moveTo(ax, ay);
            ctx.lineTo(ax - arrowSize * Math.cos(angle - Math.PI/6), ay - arrowSize * Math.sin(angle - Math.PI/6));
            ctx.lineTo(ax - arrowSize * Math.cos(angle + Math.PI/6), ay - arrowSize * Math.sin(angle + Math.PI/6));
            ctx.closePath();
            ctx.fillStyle = ctx.strokeStyle;
            ctx.fill();
        });
        
        // Draw port-to-parent connections
        portNodes.forEach(portNode => {
            const parent = nodes.get(portNode.data.parent);
            if (parent) {
                const pScreen = worldToScreen(parent.x, parent.y);
                const portScreen = worldToScreen(portNode.x, portNode.y);
                
                ctx.beginPath();
                ctx.moveTo(pScreen.x, pScreen.y);
                ctx.lineTo(portScreen.x, portScreen.y);
                ctx.strokeStyle = 'rgba(57, 186, 230, 0.2)';
                ctx.lineWidth = 1;
                ctx.stroke();
            }
        });
        
        // Draw nodes
        const allNodes = [...workloadNodes, ...portNodes];
        allNodes.forEach(node => {
            const screen = worldToScreen(node.x, node.y);
            const isHovered = hoveredNode === node;
            const isSearchMatch = searchTerm && node.data.label.toLowerCase().includes(searchTerm.toLowerCase());
            const isWorkload = node.data.type === 'workload';
            
            const radius = node.radius * zoom;
            const color = isWorkload ? colors[node.data.kind] : colors.port;
            
            // Glow effect
            if (isHovered || isSearchMatch) {
                const gradient = ctx.createRadialGradient(screen.x, screen.y, 0, screen.x, screen.y, radius * 2);
                gradient.addColorStop(0, color + '40');
                gradient.addColorStop(1, 'transparent');
                ctx.beginPath();
                ctx.arc(screen.x, screen.y, radius * 2, 0, Math.PI * 2);
                ctx.fillStyle = gradient;
                ctx.fill();
            }
            
            // Node body
            ctx.beginPath();
            if (isWorkload) {
                // Rounded rectangle for workloads
                roundRect(ctx, screen.x - radius, screen.y - radius, radius * 2, radius * 2, 8 * zoom);
            } else {
                ctx.arc(screen.x, screen.y, radius, 0, Math.PI * 2);
            }
            
            const fillGradient = ctx.createLinearGradient(screen.x - radius, screen.y - radius, screen.x + radius, screen.y + radius);
            fillGradient.addColorStop(0, color + '40');
            fillGradient.addColorStop(1, color + '20');
            ctx.fillStyle = fillGradient;
            ctx.fill();
            
            ctx.strokeStyle = isHovered ? color : color + '80';
            ctx.lineWidth = isHovered ? 2 : 1;
            ctx.stroke();
            
            // Label
            if (isWorkload || isHovered) {
                ctx.font = (isWorkload ? '600 ' : '400 ') + Math.max(10, 12 * zoom) + 'px Outfit';
                ctx.textAlign = 'center';
                ctx.textBaseline = 'middle';
                
                const label = node.data.label;
                const maxLen = 12;
                const displayLabel = label.length > maxLen ? label.slice(0, maxLen) + '…' : label;
                
                if (isWorkload) {
                    ctx.fillStyle = '#0a0e14';
                    ctx.fillText(displayLabel, screen.x, screen.y);
                    ctx.fillStyle = color;
                    ctx.fillText(displayLabel, screen.x - 0.5, screen.y - 0.5);
                } else {
                    ctx.fillStyle = color;
                    ctx.fillText(node.data.port, screen.x, screen.y);
                }
            }
            
            // Namespace badge for workloads
            if (isWorkload && zoom > 0.6) {
                ctx.font = '500 ' + Math.max(8, 9 * zoom) + 'px JetBrains Mono';
                ctx.fillStyle = 'rgba(98, 106, 115, 0.8)';
                ctx.fillText(node.data.namespace, screen.x, screen.y + radius + 12 * zoom);
            }
        });
        
        drawMinimap();
        requestAnimationFrame(draw);
    }
    
    function roundRect(ctx, x, y, w, h, r) {
        ctx.beginPath();
        ctx.moveTo(x + r, y);
        ctx.lineTo(x + w - r, y);
        ctx.quadraticCurveTo(x + w, y, x + w, y + r);
        ctx.lineTo(x + w, y + h - r);
        ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
        ctx.lineTo(x + r, y + h);
        ctx.quadraticCurveTo(x, y + h, x, y + h - r);
        ctx.lineTo(x, y + r);
        ctx.quadraticCurveTo(x, y, x + r, y);
        ctx.closePath();
    }
    
    function drawMinimap() {
        minimapCtx.clearRect(0, 0, 180, 120);
        minimapCtx.fillStyle = 'rgba(18, 24, 32, 0.9)';
        minimapCtx.fillRect(0, 0, 180, 120);
        
        // Find bounds
        let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
        workloadNodes.forEach(n => {
            minX = Math.min(minX, n.x);
            maxX = Math.max(maxX, n.x);
            minY = Math.min(minY, n.y);
            maxY = Math.max(maxY, n.y);
        });
        
        const padding = 100;
        minX -= padding; maxX += padding;
        minY -= padding; maxY += padding;
        
        const scaleX = 180 / (maxX - minX);
        const scaleY = 120 / (maxY - minY);
        const scale = Math.min(scaleX, scaleY);
        
        const offsetX = (180 - (maxX - minX) * scale) / 2;
        const offsetY = (120 - (maxY - minY) * scale) / 2;
        
        // Draw nodes
        workloadNodes.forEach(n => {
            const x = (n.x - minX) * scale + offsetX;
            const y = (n.y - minY) * scale + offsetY;
            minimapCtx.beginPath();
            minimapCtx.arc(x, y, 3, 0, Math.PI * 2);
            minimapCtx.fillStyle = colors[n.data.kind];
            minimapCtx.fill();
        });
        
        // Draw viewport rectangle
        const viewMinWorld = screenToWorld(0, 0);
        const viewMaxWorld = screenToWorld(width, height);
        
        const vx = (viewMinWorld.x - minX) * scale + offsetX;
        const vy = (viewMinWorld.y - minY) * scale + offsetY;
        const vw = (viewMaxWorld.x - viewMinWorld.x) * scale;
        const vh = (viewMaxWorld.y - viewMinWorld.y) * scale;
        
        minimapCtx.strokeStyle = 'rgba(57, 186, 230, 0.6)';
        minimapCtx.lineWidth = 1;
        minimapCtx.strokeRect(vx, vy, vw, vh);
    }
    
    function findNodeAt(x, y) {
        const world = screenToWorld(x, y);
        const allNodes = [...portNodes, ...workloadNodes]; // Check ports first
        
        for (const node of allNodes) {
            const dx = world.x - node.x;
            const dy = world.y - node.y;
            const dist = Math.sqrt(dx * dx + dy * dy);
            if (dist < node.radius + 5) {
                return node;
            }
        }
        return null;
    }
    
    function findEdgeAt(x, y) {
        const world = screenToWorld(x, y);
        
        for (const edge of edges) {
            const source = edge.sourceNode;
            const target = edge.targetNode;
            
            // Simple distance check to curved line
            const midX = (source.x + target.x) / 2;
            const midY = (source.y + target.y) / 2;
            
            const dist = Math.sqrt(Math.pow(world.x - midX, 2) + Math.pow(world.y - midY, 2));
            const lineLen = Math.sqrt(Math.pow(target.x - source.x, 2) + Math.pow(target.y - source.y, 2));
            
            if (dist < lineLen / 2) {
                // More precise check using point-to-line distance
                const A = world.x - source.x;
                const B = world.y - source.y;
                const C = target.x - source.x;
                const D = target.y - source.y;
                
                const dot = A * C + B * D;
                const lenSq = C * C + D * D;
                const param = lenSq !== 0 ? dot / lenSq : -1;
                
                let xx, yy;
                if (param < 0) {
                    xx = source.x; yy = source.y;
                } else if (param > 1) {
                    xx = target.x; yy = target.y;
                } else {
                    xx = source.x + param * C;
                    yy = source.y + param * D;
                }
                
                const dx = world.x - xx;
                const dy = world.y - yy;
                const pointDist = Math.sqrt(dx * dx + dy * dy);
                
                if (pointDist < 15) {
                    return edge;
                }
            }
        }
        return null;
    }
    
    function showTooltip(x, y, content) {
        tooltip.innerHTML = content;
        tooltip.classList.add('visible');
        
        const rect = tooltip.getBoundingClientRect();
        let left = x + 15;
        let top = y + 15;
        
        if (left + rect.width > window.innerWidth) {
            left = x - rect.width - 15;
        }
        if (top + rect.height > window.innerHeight) {
            top = y - rect.height - 15;
        }
        
        tooltip.style.left = left + 'px';
        tooltip.style.top = top + 'px';
    }
    
    function hideTooltip() {
        tooltip.classList.remove('visible');
    }
    
    function getNodeTooltip(node) {
        const data = node.data;
        if (data.type === 'workload') {
            const badgeClass = 'badge-' + data.kind.toLowerCase();
            let html = '<div class="tooltip-title">' + data.label + 
                '<span class="tooltip-badge ' + badgeClass + '">' + data.kind + '</span></div>';
            html += '<div class="tooltip-row"><span class="tooltip-label">Namespace</span><span class="tooltip-value">' + data.namespace + '</span></div>';
            html += '<div class="tooltip-row"><span class="tooltip-label">ID</span><span class="tooltip-value">' + data.id + '</span></div>';
            
            if (data.metadata) {
                const labels = Object.entries(data.metadata).slice(0, 3);
                if (labels.length > 0) {
                    html += '<div class="tooltip-row"><span class="tooltip-label">Labels</span></div>';
                    labels.forEach(([k, v]) => {
                        html += '<div class="tooltip-row" style="padding-left: 12px;"><span class="tooltip-value" style="font-size: 11px;">' + k + '=' + v + '</span></div>';
                    });
                }
            }
            return html;
        } else {
            return '<div class="tooltip-title">' + data.label + 
                '<span class="tooltip-badge badge-port">Port</span></div>' +
                '<div class="tooltip-row"><span class="tooltip-label">Port</span><span class="tooltip-value">' + data.port + '</span></div>' +
                '<div class="tooltip-row"><span class="tooltip-label">Protocol</span><span class="tooltip-value">' + data.protocol + '</span></div>' +
                '<div class="tooltip-row"><span class="tooltip-label">Parent</span><span class="tooltip-value">' + data.parent + '</span></div>';
        }
    }
    
    function getEdgeTooltip(edge) {
        let html = '<div class="tooltip-title">Network Connection</div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">From</span><span class="tooltip-value">' + edge.source + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">To</span><span class="tooltip-value">' + edge.target + '</span></div>';
        html += '<div class="tooltip-row"><span class="tooltip-label">Policy</span><span class="tooltip-value">' + edge.policy + '</span></div>';
        html += '<div class="tooltip-rule">' + edge.rule + '</div>';
        return html;
    }
    
    // Event handlers
    canvas.addEventListener('mousedown', (e) => {
        const rect = canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        const node = findNodeAt(x, y);
        if (node && node.data.type === 'workload') {
            isDragging = true;
            dragNode = node;
            dragNode.fixed = true;
            const world = screenToWorld(x, y);
            dragOffsetX = node.x - world.x;
            dragOffsetY = node.y - world.y;
        } else {
            isPanning = true;
            lastMouseX = x;
            lastMouseY = y;
        }
    });
    
    canvas.addEventListener('mousemove', (e) => {
        const rect = canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        if (isDragging && dragNode) {
            const world = screenToWorld(x, y);
            dragNode.x = world.x + dragOffsetX;
            dragNode.y = world.y + dragOffsetY;
        } else if (isPanning) {
            panX += x - lastMouseX;
            panY += y - lastMouseY;
            lastMouseX = x;
            lastMouseY = y;
        } else {
            const node = findNodeAt(x, y);
            const edge = node ? null : findEdgeAt(x, y);
            
            hoveredNode = node;
            hoveredEdge = edge;
            
            if (node) {
                showTooltip(e.clientX, e.clientY, getNodeTooltip(node));
                canvas.style.cursor = 'pointer';
            } else if (edge) {
                showTooltip(e.clientX, e.clientY, getEdgeTooltip(edge));
                canvas.style.cursor = 'pointer';
            } else {
                hideTooltip();
                canvas.style.cursor = 'grab';
            }
        }
    });
    
    canvas.addEventListener('mouseup', () => {
        if (dragNode) {
            dragNode.fixed = false;
        }
        isDragging = false;
        isPanning = false;
        dragNode = null;
    });
    
    canvas.addEventListener('mouseleave', () => {
        hideTooltip();
        if (dragNode) {
            dragNode.fixed = false;
        }
        isDragging = false;
        isPanning = false;
        dragNode = null;
    });
    
    canvas.addEventListener('wheel', (e) => {
        e.preventDefault();
        const rect = canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        const delta = e.deltaY > 0 ? 0.9 : 1.1;
        const newZoom = Math.min(Math.max(zoom * delta, 0.2), 5);
        
        // Zoom towards mouse position
        const world = screenToWorld(x, y);
        zoom = newZoom;
        panX = x - world.x * zoom;
        panY = y - world.y * zoom;
    });
    
    document.getElementById('search-input').addEventListener('input', (e) => {
        searchTerm = e.target.value;
    });
    
    function resetView() {
        panX = width / 2;
        panY = height / 2;
        zoom = 1;
    }
    
    function togglePhysics() {
        physicsEnabled = !physicsEnabled;
    }
    
    function exportPNG() {
        const link = document.createElement('a');
        link.download = 'network-map.png';
        link.href = canvas.toDataURL('image/png');
        link.click();
    }
    
    // Physics loop
    setInterval(applyPhysics, 16);
    
    // Initialize
    window.addEventListener('resize', resize);
    resize();
    draw();
    </script>
</body>
</html>`

