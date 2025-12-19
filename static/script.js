const ui = {
    // Inputs
    urlInputVideo: document.getElementById('urlInputVideo'),
    urlInputPlaylist: document.getElementById('urlInputPlaylist'),

    // Buttons
    fetchBtnVideo: document.getElementById('fetchBtnVideo'),
    fetchBtnPlaylist: document.getElementById('fetchBtnPlaylist'),

    statusMsg: document.getElementById('statusMessage'),

    // Single Video UI
    infoSection: document.getElementById('videoInfo'),
    thumb: document.getElementById('videoThumb'),
    title: document.getElementById('videoTitle'),
    duration: document.getElementById('videoDuration'),
    desc: document.getElementById('videoDesc'),
    qualitySelect: document.getElementById('qualitySelect'),
    btnMp4: document.getElementById('btnMp4'),
    btnMp3: document.getElementById('btnMp3'),

    // Playlist UI
    plSection: document.getElementById('playlistInfo'),
    plTitle: document.getElementById('plTitle'),
    plAuthor: document.getElementById('plAuthor'),
    plList: document.getElementById('plVideoList'),
    plQualitySelect: document.getElementById('plQualitySelect'),
    plProgressBar: document.getElementById('plProgressBar'),
    plProgressText: document.getElementById('plProgressText'),
    plPercent: document.getElementById('plPercent'),
    btnPlMp3: document.getElementById('btnPlMp3'),
    btnPlMp4: document.getElementById('btnPlMp4'),
    btnPlStop: document.getElementById('btnPlStop')
};

// State
let currentPlaylist = [];
let isDownloadingPlaylist = false;
let shouldStopPlaylist = false;

function showStatus(msg, type = 'info') {
    ui.statusMsg.className = `msg-toast msg-${type} d-flex`;
    ui.statusMsg.innerHTML = type === 'error'
        ? `<i class="bi bi-exclamation-triangle-fill fs-5"></i> <span>${msg}</span>`
        : (type === 'success'
            ? `<i class="bi bi-check-circle-fill fs-5"></i> <span>${msg}</span>`
            : `<i class="bi bi-info-circle-fill fs-5"></i> <span>${msg}</span>`);

    ui.statusMsg.classList.remove('d-none');
    // Auto hide only strict success
    if (type === 'success') {
        setTimeout(() => ui.statusMsg.classList.add('d-none'), 5000);
    }
}

function setLoading(type, isLoading) {
    // Determine which button to toggle
    let btn;
    if (type === 'video') btn = ui.fetchBtnVideo;
    else if (type === 'playlist') btn = ui.fetchBtnPlaylist;
    else return;

    const textSpan = btn.querySelector('.btn-text');
    const spinner = btn.querySelector('.spinner-border');
    const defaultText = type === 'video' ? 'Descargar' : 'Cargar Lista';

    btn.disabled = isLoading;
    if (isLoading) {
        textSpan.textContent = 'Cargando...';
        spinner.classList.remove('d-none');
    } else {
        textSpan.textContent = defaultText;
        spinner.classList.add('d-none');
    }
}

function setDownloadLoading(format, isLoading) {
    const btn = format === 'mp4' ? ui.btnMp4 : ui.btnMp3;
    const originalContent = format === 'mp4' ? '<i class="bi bi-film"></i> Descargar MP4' : '<i class="bi bi-music-note-beamed"></i> Descargar MP3';

    if (isLoading) {
        btn.innerHTML = `<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Procesando...`;
        btn.classList.add('btn-disabled');
        btn.disabled = true;
        if (format === 'mp4') ui.btnMp3.disabled = true;
        else ui.btnMp4.disabled = true;
    } else {
        btn.innerHTML = originalContent;
        btn.classList.remove('btn-disabled');
        btn.disabled = false;
        ui.btnMp3.disabled = false;
        ui.btnMp4.disabled = false;
    }
}

async function fetchInfo(type) {
    const input = type === 'video' ? ui.urlInputVideo : ui.urlInputPlaylist;
    const url = input.value.trim();

    if (!url) {
        showStatus('Por favor ingresa un link válido', 'error');
        return;
    }

    ui.infoSection.style.display = 'none';
    ui.plSection.style.display = 'none';
    ui.statusMsg.className = 'd-none';
    setLoading(type, true);

    try {
        const response = await fetch('/info', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: url })
        });

        if (!response.ok) throw new Error('Error al obtener info.');

        const data = await response.json();

        // Cross-check type
        if (type === 'video' && data.type === 'playlist') {
            showStatus('Es una Playlist. Usa la pestaña "Playlist / Mix".', 'info');
            setLoading(type, false);
            return;
        }
        if (type === 'playlist' && data.type === 'video') {
            showStatus('Es un Video individual. Usa la pestaña "Video".', 'info');
            setLoading(type, false);
            return;
        }

        if (data.type === 'playlist') {
            renderPlaylist(data.playlist);
        } else if (data.type === 'video') {
            renderVideo(data.video);
        } else {
            if (data.title) renderVideo(data);
            else throw new Error('Formato desconocido');
        }

    } catch (err) {
        showStatus(err.message, 'error');
    } finally {
        setLoading(type, false);
    }
}

function renderVideo(data) {
    ui.thumb.src = data.thumbnail;
    ui.title.textContent = data.title;
    ui.duration.innerHTML = `<i class="bi bi-clock me-1"></i> ${data.duration}`;
    ui.desc.textContent = data.description || "Sin descripción.";

    ui.qualitySelect.innerHTML = '<option value="">Selecciona calidad...</option>';
    data.formats.forEach(f => {
        const opt = document.createElement('option');
        opt.value = f.itag;
        const label = f.label || "Unknown";
        opt.textContent = `${label} (${f.type.split(';')[0]})`;
        ui.qualitySelect.appendChild(opt);
    });
    if (ui.qualitySelect.options.length > 1) ui.qualitySelect.selectedIndex = 1;

    ui.infoSection.style.display = 'block';
}

function renderPlaylist(playlist) {
    ui.plTitle.textContent = playlist.title;
    ui.plAuthor.textContent = `by ${playlist.author}`;
    currentPlaylist = playlist.videos;

    ui.plList.innerHTML = '';
    playlist.videos.forEach((v, index) => {
        const item = document.createElement('div');
        item.className = 'list-group-item d-flex justify-content-between align-items-center';
        item.id = `pl-item-${index}`;
        item.innerHTML = `
            <div class="d-flex align-items-center gap-3">
                <span class="text-muted small fw-bold">#${index + 1}</span>
                <div class="text-truncate" style="max-width: 250px;">${v.title}</div>
            </div>
            <span class="badge bg-light text-dark status-badge">Pendiente</span>
        `;
        ui.plList.appendChild(item);
    });

    resetPlaylistProgress();
    ui.plSection.style.display = 'block';
}

function resetPlaylistProgress() {
    ui.plProgressBar.style.width = '0%';
    ui.plProgressText.textContent = `0 / ${currentPlaylist.length} Completados`;
    ui.plPercent.textContent = '0%';
    shouldStopPlaylist = false;
    isDownloadingPlaylist = false;
    ui.btnPlStop.disabled = true;
    ui.btnPlMp3.disabled = false;
    ui.btnPlMp4.disabled = false;
}

async function downloadPlaylist(format) {
    if (isDownloadingPlaylist) return;
    isDownloadingPlaylist = true;
    shouldStopPlaylist = false;
    ui.btnPlStop.disabled = false;
    ui.btnPlMp3.disabled = true;
    ui.btnPlMp4.disabled = true;

    let completed = 0;
    const total = currentPlaylist.length;

    for (let i = 0; i < total; i++) {
        if (shouldStopPlaylist) {
            showStatus('Descarga detenida.', 'info');
            break;
        }

        const video = currentPlaylist[i];
        const itemEl = document.getElementById(`pl-item-${i}`);
        const badge = itemEl.querySelector('.status-badge');

        badge.className = 'badge bg-primary text-white status-badge';
        badge.innerHTML = '<span class="spinner-border spinner-border-sm" style="width: 0.7rem; height: 0.7rem;"></span> DL';
        itemEl.scrollIntoView({ behavior: 'smooth', block: 'center' });

        try {
            const vidUrl = `https://www.youtube.com/watch?v=${video.id}`;
            const quality = ui.plQualitySelect.value;

            const response = await fetch('/download', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url: vidUrl, format: format, quality: quality })
            });

            if (!response.ok) throw new Error('Failed');

            const blob = await response.blob();
            const downloadUrl = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = downloadUrl;

            const disposition = response.headers.get('Content-Disposition');
            let filename = `${video.title}.${format}`;
            if (disposition && disposition.indexOf('filename=') !== -1) {
                const matches = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/.exec(disposition);
                if (matches != null && matches[1]) filename = matches[1].replace(/['"]/g, '');
            }

            a.download = filename;
            document.body.appendChild(a);
            a.click();
            a.remove();
            window.URL.revokeObjectURL(downloadUrl);

            badge.className = 'badge bg-success text-white status-badge';
            badge.textContent = 'Listo';

        } catch (err) {
            console.error(err);
            badge.className = 'badge bg-danger text-white status-badge';
            badge.textContent = 'Error';
        }

        completed++;
        const percent = Math.round((completed / total) * 100);
        ui.plProgressBar.style.width = `${percent}%`;
        ui.plProgressText.textContent = `${completed} / ${total} Completados`;
        ui.plPercent.textContent = `${percent}%`;
    }

    isDownloadingPlaylist = false;
    ui.btnPlStop.disabled = true;
    ui.btnPlMp3.disabled = false;
    ui.btnPlMp4.disabled = false;
    if (!shouldStopPlaylist) showStatus('Playlist completada!', 'success');
}

function stopPlaylist() {
    shouldStopPlaylist = true;
    ui.btnPlStop.disabled = true;
}

async function download(format) {
    const url = ui.urlInputVideo.value.trim();
    let quality = "";

    if (format === 'mp4') {
        quality = ui.qualitySelect.value;
        if (!quality) {
            showStatus('Por favor selecciona una calidad.', 'error');
            return;
        }
    }

    showStatus(`Iniciando descarga ${format.toUpperCase()}...`, 'info');
    setDownloadLoading(format, true);

    try {
        const response = await fetch('/download', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url: url, format: format, quality: quality })
        });

        if (!response.ok) {
            const text = await response.text();
            throw new Error(text || 'Download failed');
        }

        const blob = await response.blob();
        const downloadUrl = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = downloadUrl;

        const disposition = response.headers.get('Content-Disposition');
        let filename = `video.${format}`;
        if (disposition && disposition.indexOf('filename=') !== -1) {
            const matches = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/.exec(disposition);
            if (matches != null && matches[1]) {
                filename = matches[1].replace(/['"]/g, '');
            }
        }

        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(downloadUrl);

        showStatus('Descarga iniciada exitosamente!', 'success');

    } catch (err) {
        console.error(err);
        showStatus(err.message, 'error');
    } finally {
        setDownloadLoading(format, false);
    }
}

// Trigger fetch on Enter key
ui.urlInputVideo.addEventListener('keypress', function (e) {
    if (e.key === 'Enter') fetchInfo('video');
});
ui.urlInputPlaylist.addEventListener('keypress', function (e) {
    if (e.key === 'Enter') fetchInfo('playlist');
});
