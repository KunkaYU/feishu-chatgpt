package openai

import (
	"fmt"
	"start-feishubot/initialization"
	"testing"
)

func TestCompletions(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")

	msgs := []Messages{
		{Role: "system", Content: "你是一个专业的翻译官，负责中英文翻译。"},
		{Role: "user", Content: "翻译这段话: The assistant messages help store prior responses. They can also be written by a developer to help give examples of desired behavior."},
	}

	gpt := NewChatGPT(*config)

	resp, err := gpt.Completions(msgs)
	if err != nil {
		t.Errorf("TestCompletions failed with error: %v", err)
	}

	fmt.Println(resp.Content, resp.Role)
}

func TestGenerateOneImage(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")
	gpt := NewChatGPT(*config)
	prompt := "a red apple"
	size := "256x256"
	imageURL, err := gpt.GenerateOneImage(prompt, size)
	if err != nil {
		t.Errorf("TestGenerateOneImage failed with error: %v", err)
	}
	if imageURL == "" {
		t.Errorf("TestGenerateOneImage returned empty imageURL")
	}
}

func TestAudioToText(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")
	gpt := NewChatGPT(*config)
	audio := "./test_file/test.wav"
	text, err := gpt.AudioToText(audio)
	if err != nil {
		t.Errorf("TestAudioToText failed with error: %v", err)
	}
	fmt.Printf("TestAudioToText returned text: %s \n", text)
	if text == "" {
		t.Errorf("TestAudioToText returned empty text")
	}

}

func TestVariateOneImage(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")
	gpt := NewChatGPT(*config)
	image := "./test_file/img.png"
	size := "256x256"
	//compressionType, err := GetImageCompressionType(image)
	//if err != nil {
	//	return
	//}
	//fmt.Println("compressionType: ", compressionType)
	ConvertToRGBA(image, image)
	err := VerifyPngs([]string{image})
	if err != nil {
		t.Errorf("TestVariateOneImage failed with error: %v", err)
		return
	}

	imageBs64, err := gpt.GenerateOneImageVariation(image, size)
	if err != nil {
		t.Errorf("TestVariateOneImage failed with error: %v", err)
	}
	//fmt.Printf("TestVariateOneImage returned imageBs64: %s \n", imageBs64)
	if imageBs64 == "" {
		t.Errorf("TestVariateOneImage returned empty imageURL")
	}
}

func TestVariateOneImageWithJpg(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")
	gpt := NewChatGPT(*config)
	image := "./test_file/test.jpg"
	size := "256x256"
	compressionType, err := GetImageCompressionType(image)
	if err != nil {
		return
	}
	fmt.Println("compressionType: ", compressionType)
	//ConvertJPGtoPNG(image)
	ConvertToRGBA(image, image)
	err = VerifyPngs([]string{image})
	if err != nil {
		t.Errorf("TestVariateOneImage failed with error: %v", err)
		return
	}

	imageBs64, err := gpt.GenerateOneImageVariation(image, size)
	if err != nil {
		t.Errorf("TestVariateOneImage failed with error: %v", err)
	}
	fmt.Printf("TestVariateOneImage returned imageBs64: %s \n", imageBs64)
	if imageBs64 == "" {
		t.Errorf("TestVariateOneImage returned empty imageURL")
	}
}

func TestSQLCompletions(t *testing.T) {
	config := initialization.LoadConfig("../../config.yaml")

	msgs := []Messages{
		{Role: "system", Content: "你是一个SQL语句生成器，负责帮我生成SQL语句，语句基于Postgres语法。表结构信息如下："},
		{Role: "system", Content: "ac_crawler_contract_prices表中有所有token的价格，包含列有：quote_currency_smart_contract(string)token合约地址，asset_name(string)token的名称，chain(string)token所在的链，price_timestamp(string)yyyy-MM-dd HH:mm:ss格式的时间戳，close(float)价格；"},
		{Role: "system", Content: "bsc_ads.ads_addr_balance表中有所有用户地址所有token的持仓余额，包含的列有：org_address(string)用户地址，contract_address(string)token合约地址，token_decimals(int)表示余额中小数点右移的位数，balance(numeric)表示地址在该币种的小数点右移token_decimals位后的持仓；"},

		{Role: "assistant", Content: "eth_dim.dim_addr_contracts每个合约一条记录，包含如下列：contract_address(string)合约地址，deployer（string）部署合约的地址，block_timestamp（bigint）合约的部署时间；"},
		{Role: "assistant", Content: "eth_dim.dim_addr_deposit_addresses每个充币地址一条记录，包含如下列：address（string）充币地址，exchange_name（string）充币地址所属交易所的名称"},
		{Role: "user", Content: "生成这个查询SQL: 查询哪个地址的总资产最多"},
	}

	gpt := NewChatGPT(*config)

	resp, err := gpt.Completions(msgs)
	if err != nil {
		t.Errorf("TestCompletions failed with error: %v", err)
	}

	fmt.Println(resp.Content, resp.Role)
}
