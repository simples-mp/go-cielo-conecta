package go_cielo_conecta

import (
	"strings"
)

// Essa configuração inicial é essencial para que você obtenha todas as tabelas de parâmetros necessárias para que a
// solução de captura funcione corretamente via API. Os dados são fornecidos por meio de uma chamada à API e devem ser
// instalados na Biblioteca Compartilhada (BC) do terminal.
//
// Esses parâmetros incluem informações como bandeiras, emissores, produtos, regras de negócio e configurações técnicas,
// garantindo que o terminal esteja apto a realizar transações de forma segura e conforme as especificações da Cielo.
//
// GET /api/v0.1/initialization/:SubordinatedMerchantId/:TerminalId
func (c *Client) SharedLibrary(terminalID string, subMerchantId ...string) (map[string]any, error) {
	var (
		path string
		data map[string]any
	)

	path = strings.Replace(c.env.ParamsURL, "{SubordinatedMerchantId}", c.env.merchant.ID, 1)

	if len(subMerchantId) > 0 {
		path = strings.Replace(c.env.ParamsURL, "{SubordinatedMerchantId}", subMerchantId[0], 1)
	}

	path = strings.Replace(path, "{TerminalId}", terminalID, 1)

	req, err := c.NewRequest("GET", path, nil)
	if err != nil {
		return data, err
	}

	err = c.Send(req, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}
