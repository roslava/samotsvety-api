// internal/imaging/webp.go
package imaging

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ToWebP конвертирует произвольное растровое изображение (jpeg/png/webp/gif-первый кадр)
// в WebP через бинарник cwebp (libwebp).
//
// Почему exec.Command(cwebp), а не чистый Go или cgo-биндинг:
//   - чистые Go WebP-энкодеры дают заметно худшее сжатие/качество, чем libwebp;
//   - cgo-биндинги (chai2010/webp, kolesa-team/go-webp) требуют libwebp-dev на
//     этапе СБОРКИ и усложняют кросс-компиляцию;
//   - cwebp как внешний бинарник нужен только на этапе ВЫПОЛНЕНИЯ — один apt-пакет
//     на сервере (`apt-get install webp`), без влияния на сборку Go-бинаря.
//
// quality — 0..100. Для каталожных фото минералов используем высокое значение
// (по умолчанию 90) без изменения разрешения — cwebp сам не апскейлит/даунскейлит,
// если не передавать -resize.
func ToWebP(ctx context.Context, data []byte, quality int) ([]byte, error) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		return nil, fmt.Errorf("imaging: бинарник cwebp не найден в PATH — установите пакет webp (apt-get install webp) на сервере: %w", err)
	}

	if quality <= 0 || quality > 100 {
		quality = 90
	}

	tmpDir, err := os.MkdirTemp("", "webp-convert-*")
	if err != nil {
		return nil, fmt.Errorf("imaging: не удалось создать временную директорию: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inPath := filepath.Join(tmpDir, "in")
	outPath := filepath.Join(tmpDir, "out.webp")

	if err := os.WriteFile(inPath, data, 0o600); err != nil {
		return nil, fmt.Errorf("imaging: не удалось записать временный файл: %w", err)
	}

	// -q <quality>  — качество сжатия (без -lossless: для фото лучше баланс размер/резкость)
	// -m 6          — самый медленный, но самый качественный метод сжатия
	// -mt           — многопоточное кодирование
	// -quiet        — не засорять логи
	cmd := exec.CommandContext(ctx, "cwebp",
		"-q", fmt.Sprintf("%d", quality),
		"-m", "6",
		"-mt",
		"-quiet",
		inPath,
		"-o", outPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("imaging: cwebp завершился с ошибкой: %w (%s)", err, stderr.String())
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("imaging: не удалось прочитать результат конвертации: %w", err)
	}

	return out, nil
}
