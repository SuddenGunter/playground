#include <string.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_lcd_panel_io.h"
#include "esp_lcd_panel_ops.h"
#include "esp_lcd_ili9341.h"
#include "driver/gpio.h"
#include "driver/spi_master.h"
#include "esp_random.h"

#define PIN_NUM_SCLK      12
#define PIN_NUM_MOSI      11
#define PIN_NUM_MISO      13
#define PIN_NUM_LCD_DC    9
#define PIN_NUM_LCD_RST   14
#define PIN_NUM_LCD_CS    10

// Backlight/LED is hardwired to 3.3V, not controlled by GPIO
// #define PIN_NUM_BK_LIGHT  2    // Not used – backlight is always on when 3.3V is applied

#define LCD_HOST          SPI2_HOST
#define LCD_H_RES         240
#define LCD_V_RES         320

#define BOX_SIZE          100   // width and height of the moving box

void app_main(void)
{
    // 1. Initialize SPI Bus
    spi_bus_config_t buscfg = {
        .sclk_io_num = PIN_NUM_SCLK,
        .mosi_io_num = PIN_NUM_MOSI,
        .miso_io_num = PIN_NUM_MISO,
        .data2_io_num = -1,    // v6: replaced quadwp_io_num
        .data3_io_num = -1,    // v6: replaced quadhd_io_num
        .max_transfer_sz = LCD_H_RES * 80 * sizeof(uint16_t),
    };
    ESP_ERROR_CHECK(spi_bus_initialize(LCD_HOST, &buscfg, SPI_DMA_CH_AUTO));

    // 2. Attach LCD controller to SPI bus (Panel IO)
    esp_lcd_panel_io_handle_t io_handle = NULL;
    esp_lcd_panel_io_spi_config_t io_config = {
        .dc_gpio_num = PIN_NUM_LCD_DC,
        .cs_gpio_num = PIN_NUM_LCD_CS,
        .pclk_hz = 26 * 1000 * 1000,  // 26 MHz SPI Clock
        .lcd_cmd_bits = 8,
        .lcd_param_bits = 8,
        .spi_mode = 0,
        .trans_queue_depth = 10,
    };
    ESP_ERROR_CHECK(
        esp_lcd_new_panel_io_spi((esp_lcd_spi_bus_handle_t)LCD_HOST, &io_config, &io_handle));

    // 3. Install ILI9341 Panel Driver
    esp_lcd_panel_handle_t panel_handle = NULL;
    esp_lcd_panel_dev_config_t panel_config = {
        .reset_gpio_num = PIN_NUM_LCD_RST,
        .rgb_ele_order = LCD_RGB_ELEMENT_ORDER_RGB,   // Use RGB order for standard RGB565
        .bits_per_pixel = 16,
    };
    ESP_ERROR_CHECK(esp_lcd_new_panel_ili9341(io_handle, &panel_config, &panel_handle));

    // 4. Reset & Initialize Display
    ESP_ERROR_CHECK(esp_lcd_panel_reset(panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_init(panel_handle));
    ESP_ERROR_CHECK(esp_lcd_panel_disp_on_off(panel_handle, true));

    // 5. Backlight is hardwired to 3.3V – no GPIO control needed

    // Allocate buffers once
    uint16_t *clear_buf = malloc(LCD_H_RES * sizeof(uint16_t));
    uint16_t *box_buf  = malloc(BOX_SIZE * BOX_SIZE * sizeof(uint16_t));
    assert(clear_buf != NULL && box_buf != NULL);

    // Pre-fill clear buffer with black (0x0000)
    memset(clear_buf, 0x00, LCD_H_RES * sizeof(uint16_t));

    // Animation loop: change position and color every 1 second
    while (1) {
        // Generate random top-left coordinates within screen bounds
        int x_start = esp_random() % (LCD_H_RES - BOX_SIZE + 1);
        int y_start = esp_random() % (LCD_V_RES - BOX_SIZE + 1);

        // Generate random RGB565 color (16-bit)
        uint16_t color = (uint16_t)(esp_random() & 0xFFFF);

        // Fill box buffer with the random color
        for (int i = 0; i < BOX_SIZE * BOX_SIZE; i++) {
            box_buf[i] = color;
        }

        // Clear entire screen to black
        for (int y = 0; y < LCD_V_RES; y++) {
            esp_lcd_panel_draw_bitmap(panel_handle, 0, y, LCD_H_RES, y + 1, clear_buf);
        }

        // Draw the box at the new random position
        esp_lcd_panel_draw_bitmap(panel_handle,
                                  x_start, y_start,
                                  x_start + BOX_SIZE, y_start + BOX_SIZE,
                                  box_buf);

        // Wait 1 second before the next move
        vTaskDelay(pdMS_TO_TICKS(1000));
    }

    // The loop never exits, but free buffers here for completeness
    free(clear_buf);
    free(box_buf);
}