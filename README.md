# Описание qBox
Консольная утилита qBox предоставляет возможность опроса различных теплосчётчиков и вывод полученных данных.
Для опроса теплосчётчиков используются различные драйверы, которые написаны в соответствии с протоколами обмена 
теплосчётчиков.

Работа утилиты qBox описана в руководстве, которое можно увидеть выполнив команду в консоле:
```bash
qBox -h
```

# Сборка программы
Для успешной компиляции, сборки необходимо установить golang версии не ниже `1.9.0`.
Затем выполнить команду для компиляции в директории с `main.go`

- для `WIN32` выполнить `set GOARCH=386&&set GOOS=windows&&go build`
- для `WIN64` выполнить `set GOARCH=amd64&&set GOOS=windows&&go build`
- для `Linux32` выполнить `set GOARCH=386&&set GOOS=linux&&go build`
- для `Linux64` выполнить `set GOARCH=amd64&&set GOOS=linux&&go build`


# Создание драйвера

Создание нового драйвера для опроса теплосчётчика требует:
- создание файла с расширением go в отдельной директории
- реализация интерфейса `models/driver.go`

## Создание файла

Файлы драйвера следует располагать в директории `drivers/drivername/driver.go`, где `drivername` имя драйвера на латинице.
Например для драйвера ТЭМ-104, файл драйвера следует создать по пути `drivers/tem104/driver.go`

## Реализация интерфейса драйвера

Для того, чтобы драйвер работал корректно необходимо реализовать интерфейс `models/DriverInterface`.
Например начальный код драйвера для того же ТЭМ-104 может выглядеть так:

```go
package tem104

import (
	"qBox/models"
	"qBox/services/log"
	"qBox/services/net"
) 

type Driver struct {
	data          models.DataDevice
	network       *net.Network
	logger        *log.LoggerService
	counterNumber byte
}

// Реализация интерфейса DriverInterface::Init
func (driver *Driver) Init(counterNumber byte, network *net.Network, logger *log.LoggerService) error {
	driver.logger = logger
	driver.network = network
	driver.counterNumber = counterNumber
	return nil
}

// Реализация интерфейса DriverInterface::Read
func (driver *Driver) Read() (*models.DataDevice, error) {
	return &driver.data, nil
}
```

В методе `Driver.Init` следует:
 - получать техническую информацию, которая в дальнейшем позволяет получить текущие данные с минимальными затратами.
- задавать коэффициенты перевода единиц измерения энергии

В случае проблем с инициализацией должна возвращаться ошибка.

В методе `Driver.Read` реализуется чтение текущих данных. После выполнения полученные данные должны быть заполнены
согласно структуре DataDevice. В случае безуспешного чтения, должна быть возвращена ошибка и структура данных DataDevice.

Примечание: DataDevice лучше возвращать всегда, так как ошибка может возникнуть на середине процесса 
чтения данных, но при этом хоть какая-то их часть была прочитана и этих данных, возможно, достаточно пользователю.

После создания нового драйвера, нужно добавить его в карту драйверов в файле `services/config/config.go`
```
var driversMap = [1]models.DriverInterface{
	new(drivers.tem104.Driver)}
```