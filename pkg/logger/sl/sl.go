package sl

import "log/slog"

// Err создаёт slog.Attr для логирования ошибки с ключом "error".
func Err(err error) slog.Attr {
	return slog.String("error", err.Error())
}

// Op создаёт slog.Attr с именем текущей операции.
func Op(op string) slog.Attr {
	return slog.String("op", op)
}
