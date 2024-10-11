package drivers

import (
	"qBox/models"
	"qBox/services/log"
	"qBox/services/net"
	"strconv"
	"time"
)

// Драйвер согласно протоколу ТЭМ-206
type TEM206 struct {
	data          models.DataDevice
	network       *net.Network
	logger        *log.LoggerService
	counterNumber byte
}

// Инициализация прибора
func (tem *TEM206) Init(counterNumber byte, network *net.Network, logger *log.LoggerService) error {
	tem.logger = logger
	tem.network = network
	tem.counterNumber = counterNumber

	var command []byte
	var response []byte
	var err error

	tem.logger.Info("Инициализация прибора, № %d", tem.counterNumber)
	tem.logger.Info("Читаем заводской номер")
	command = []byte{0xAA, tem.counterNumber, ToNotByte(tem.counterNumber), 0x01, 0x01, 0x00, 0x00}
	request := net.PrepareRequest(append(command, tem.calculateCheckSum(command)))
	request.SecondsReadTimeout = 5
	response, err = tem.network.RunIO(request)
	if err != nil {
		return err
	}

	tem.data.AddNewSystem(1)
	tem.data.Systems[0].Status = true
	tem.data.UnitQ = models.Gcal

	tem.data.Serial = strconv.FormatUint(uint64(calculateLongByPointerLittleEndian(response, 0x06)), 10)
	tem.logger.Debug("Байты заводского номера (%s) - %X", tem.data.Serial, response[6:10])
	return nil
}

// Чтение данных с устройства
func (tem *TEM206) Read() (*models.DataDevice, error) {
	tem.data.TimeRequest = time.Now()

	var command []byte
	var response []byte
	var err error

	// Получение даты и времени
	tem.logger.Info("Получение даты времени на теплосчётчике")
	command = []byte{0xAA, tem.counterNumber, ToNotByte(tem.counterNumber), 0x01, 0x02, 0x00, 0x00}
	request := net.PrepareRequest(append(command, tem.calculateCheckSum(command)))
	request.ControlFunction = tem.checkFrame
	request.SecondsReadTimeout = 5
	response, err = tem.network.RunIO(request)
	if err != nil {
		return &tem.data, err
	}

	year := 2000 + int(response[11])
	month := time.Month(int(response[10]))
	day := int(response[9])
	hour := int(response[8])
	min := int(response[7])
	sec := int(response[6])
	tem.data.Time = time.Date(year, month, day, hour, min, sec, 0, time.Local)

	// Чтение настроек системы
	tem.logger.Info("Чтение настроек системы")
	command = []byte{0xAA, tem.counterNumber, ToNotByte(tem.counterNumber), 0x0C, 0x01, 0x03, 0x00, 0x00, 0x28}
	request = net.PrepareRequest(append(command, tem.calculateCheckSum(command)))
	response, err = tem.network.RunIO(request)
	if err != nil {
		return &tem.data, err
	}

	tem.data.Systems[0].GV1 = calculateFloatByPointerLittleEndian(response, 0x06+0x20)
	tem.data.Systems[0].GM1 = calculateFloatByPointerLittleEndian(response, 0x06+0x24)

	// Чтение архивных данных (интеграторов)
	tem.logger.Info("Чтение архивных данных")
	command = []byte{0xAA, tem.counterNumber, ToNotByte(tem.counterNumber), 0x01, 0x03, 0x01, 0x80, 0x51}
	request = net.PrepareRequest(append(command, tem.calculateCheckSum(command)))
	response, err = tem.network.RunIO(request)
	if err != nil {
		return &tem.data, err
	}

	tem.data.Systems[0].SigmaQ = float64(calculateLongByPointerLittleEndian(response, 0x06+0x10))
	tem.data.Systems[0].V1 = float64(calculateLongByPointerLittleEndian(response, 0x06+0x08))
	tem.data.Systems[0].M1 = float64(calculateLongByPointerLittleEndian(response, 0x06+0x0C))
	tem.data.Systems[0].T1 = float32(toWord([2]byte{response[0x06+0x4B+0x01], response[0x06+0x4B]})) / 100
	tem.data.Systems[0].T2 = float32(toWord([2]byte{response[0x06+0x4B+0x03], response[0x06+0x4B+0x02]})) / 100
	tem.data.Systems[0].P1 = float32(response[0x06+0x4F]) / 100
	tem.data.Systems[0].P2 = float32(response[0x06+0x50]) / 100

	tem.data.TimeOn = calculateLongByPointerLittleEndian(response, 0x06+0x28)
	tem.data.Systems[0].TimeRunSys = calculateLongByPointerLittleEndian(response, 0x06+0x30)

	return &tem.data, nil
}

// Проверка контрольной суммы
func (*TEM206) calculateCheckSum(bytes []byte) byte {
	var sum byte = 0
	for _, b := range bytes {
		sum += b
	}
	return ^(sum & 0xFF)
}

func (tem *TEM206) checkFrame(response []byte) bool {
	if len(response) < 6 {
		tem.logger.Info("Получено меньше 6 байт")
		return false
	}

	if response[0] != 0xAA || ((^response[1])&0xFF) != response[2] {
		tem.logger.Info("Заголовок ответа не верный: %X", response[:5])
		return false
	}

	if len(response) < 6+int(response[5])+1 {
		tem.logger.Info("Размер полученных данных меньше ожидаемого: %X", 6+int(response[5])+1)
		return false
	}

	checkSum := response[len(response)-1]
	calculatedCheckSum := tem.calculateCheckSum(response[:len(response)-1])
	if calculatedCheckSum != checkSum {
		tem.logger.Info("Получен некорректный ответ. Контрольная сумма не совпадает.")
		tem.logger.Debug("Ожидалась контрольная сумма- %X", calculatedCheckSum)
		tem.logger.Debug("Получена контрольная сумма- %X", checkSum)
		return false
	}

	return true
}
